package gamerecord

import (
	"context"
	"encoding/json"
	"fmt"

	gamerecordrepo "github.com/kaifa/game-platform/internal/repository/gamerecord"
	"github.com/kaifa/game-platform/pkg/models"
)

type Service interface {
	GetUserRecords(ctx context.Context, userID uint, gameType string, page, pageSize int) ([]*GameRecordResponse, int64, error)
	GetRecordDetail(ctx context.Context, recordID uint, userID uint) (*GameRecordDetailResponse, error)
	GetRoomRecords(ctx context.Context, roomID string, userID uint) ([]*GameRecordResponse, error)
}

type service struct {
	repo gamerecordrepo.Repository
}

func New(repo gamerecordrepo.Repository) Service {
	return &service{repo: repo}
}

func (s *service) GetUserRecords(ctx context.Context, userID uint, gameType string, page, pageSize int) ([]*GameRecordResponse, int64, error) {
	roomIDs, err := s.repo.ListRoomIDsByUser(ctx, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("查询房间ID失败: %w", err)
	}
	if len(roomIDs) == 0 {
		return []*GameRecordResponse{}, 0, nil
	}
	total, err := s.repo.CountRecordsByRoomIDs(ctx, roomIDs, gameType)
	if err != nil {
		return nil, 0, fmt.Errorf("查询总数失败: %w", err)
	}
	if total == 0 {
		return []*GameRecordResponse{}, 0, nil
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	records, err := s.repo.ListRecordsByRoomIDs(ctx, roomIDs, gameType, offset, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("查询游戏记录失败: %w", err)
	}
	result := make([]*GameRecordResponse, 0, len(records))
	for i := range records {
		recordResp, err := buildRecordResponse(&records[i], userID)
		if err != nil {
			continue
		}
		result = append(result, recordResp)
	}
	return result, total, nil
}

func (s *service) GetRecordDetail(ctx context.Context, recordID uint, userID uint) (*GameRecordDetailResponse, error) {
	record, err := s.repo.GetRecordByID(ctx, recordID)
	if err != nil {
		return nil, fmt.Errorf("查询游戏记录失败: %w", err)
	}
	if _, err := s.repo.GetPlayerInRoom(ctx, record.RoomID, userID); err != nil {
		return nil, fmt.Errorf("你没有参与该游戏: %w", err)
	}
	room, err := s.repo.GetRoomByRoomID(ctx, record.RoomID)
	if err != nil {
		return nil, fmt.Errorf("查询房间失败: %w", err)
	}
	players, err := s.repo.ListPlayersByRoom(ctx, record.RoomID)
	if err != nil {
		return nil, fmt.Errorf("查询玩家失败: %w", err)
	}
	recordResp, err := buildRecordResponse(record, userID)
	if err != nil {
		return nil, fmt.Errorf("构建记录响应失败: %w", err)
	}
	detail := &GameRecordDetailResponse{
		Record:  *recordResp,
		Room:    *room,
		Players: make([]PlayerRecordResponse, 0, len(players)),
	}

	var resultData map[string]interface{}
	if len(record.Result) > 0 {
		_ = json.Unmarshal(record.Result, &resultData)
	}
	for _, player := range players {
		playerResp := PlayerRecordResponse{
			UserID:   player.UserID,
			Position: player.Position,
			Balance:  player.Balance,
		}
		if resultData != nil {
			if playerResult, ok := resultData[fmt.Sprintf("%d", player.UserID)].(map[string]interface{}); ok {
				if rank, ok := playerResult["rank"].(float64); ok {
					playerResp.Rank = int(rank)
				}
			}
		}
		detail.Players = append(detail.Players, playerResp)
	}
	return detail, nil
}

func (s *service) GetRoomRecords(ctx context.Context, roomID string, userID uint) ([]*GameRecordResponse, error) {
	if _, err := s.repo.GetPlayerInRoom(ctx, roomID, userID); err != nil {
		return nil, fmt.Errorf("你没有参与该房间: %w", err)
	}
	records, err := s.repo.ListRecordsByRoom(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("查询房间记录失败: %w", err)
	}
	result := make([]*GameRecordResponse, 0, len(records))
	for i := range records {
		recordResp, err := buildRecordResponse(&records[i], userID)
		if err != nil {
			continue
		}
		result = append(result, recordResp)
	}
	return result, nil
}

func buildRecordResponse(record *models.GameRecord, userID uint) (*GameRecordResponse, error) {
	resp := &GameRecordResponse{
		ID:          record.ID,
		RoomID:      record.RoomID,
		GameType:    record.GameType,
		StartTime:   record.StartTime,
		EndTime:     record.EndTime,
		Duration:    record.Duration,
		CreatedAt:   record.CreatedAt,
		PlayerCount: 0,
		MyRank:      0,
		MyBalance:   0,
	}
	var playersData []map[string]interface{}
	if len(record.Players) > 0 {
		if err := json.Unmarshal(record.Players, &playersData); err == nil {
			resp.PlayerCount = len(playersData)
		}
	}
	var resultData map[string]interface{}
	if len(record.Result) > 0 {
		if err := json.Unmarshal(record.Result, &resultData); err == nil {
			userIDStr := fmt.Sprintf("%d", userID)
			if userResult, ok := resultData[userIDStr].(map[string]interface{}); ok {
				if rankVal, ok := userResult["rank"]; ok {
					switch v := rankVal.(type) {
					case float64:
						resp.MyRank = int(v)
					case int:
						resp.MyRank = v
					case int64:
						resp.MyRank = int(v)
					}
				}
				if balanceVal, ok := userResult["balance"]; ok {
					switch v := balanceVal.(type) {
					case float64:
						resp.MyBalance = v
					case int:
						resp.MyBalance = float64(v)
					case int64:
						resp.MyBalance = float64(v)
					}
				}
			}
		}
	}
	return resp, nil
}

type GameRecordResponse struct {
	ID          uint    `json:"id"`
	RoomID      string  `json:"room_id"`
	GameType    string  `json:"game_type"`
	PlayerCount int     `json:"player_count"`
	MyRank      int     `json:"my_rank"`
	MyBalance   float64 `json:"my_balance"`
	StartTime   int64   `json:"start_time"`
	EndTime     int64   `json:"end_time"`
	Duration    int     `json:"duration"`
	CreatedAt   int64   `json:"created_at"`
}

type GameRecordDetailResponse struct {
	Record  GameRecordResponse     `json:"record"`
	Room    models.GameRoom        `json:"room"`
	Players []PlayerRecordResponse `json:"players"`
}

type PlayerRecordResponse struct {
	UserID   uint    `json:"user_id"`
	Position int     `json:"position"`
	Rank     int     `json:"rank"`
	Balance  float64 `json:"balance"`
}
