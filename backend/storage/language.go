package storage

import (
	"CanMe/backend/consts"
	stringutil "CanMe/backend/utils/stringUtil"
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
)

type LanguageData struct {
	gorm.Model
	Label    string `gorm:"type:varchar(36);primaryKey;not null" json:"label"`
	Value    string `gorm:"type:varchar(36);not null" json:"value"`
	Disabled bool   `gorm:"type:boolean;not null" json:"disabled"`
	Group    string `gorm:"type:varchar(36);not null" json:"group"`
}

type Language struct {
	Langs []GroupLanguage `json:"langs"`
}

type GroupLanguage struct {
	Type     string                   `json:"type"`
	Key      consts.LanguageGroupType `json:"key"`
	Label    consts.LanguageGroupType `json:"label"`
	Children []GroupLanguageChildren  `json:"children"`
}

type GroupLanguageChildren struct {
	Label    string `json:"label"`
	Value    string `json:"value"`
	Disabled bool   `json:"disabled"`
	Group    string `json:"group"`
}

func NewLanguage() *Language {
	return &Language{
		Langs: []GroupLanguage{
			{
				Type:  "group",
				Key:   consts.LANGUAGE_GROUP_TYPE_COMMON,
				Label: consts.LANGUAGE_GROUP_TYPE_COMMON,
			},
			{
				Type:  "group",
				Key:   consts.LANGUAGE_GROUP_TYPE_EXTRA,
				Label: consts.LANGUAGE_GROUP_TYPE_EXTRA,
			},
		},
	}
}

func DefaultLanguage() *Language {
	return &Language{
		Langs: []GroupLanguage{
			{
				Type:  "group",
				Key:   consts.LANGUAGE_GROUP_TYPE_COMMON,
				Label: consts.LANGUAGE_GROUP_TYPE_COMMON,
				Children: []GroupLanguageChildren{
					{
						Label: "English",
						Value: "English",
					},
					{
						Label: "Japanese",
						Value: "Japanese",
					},
					{
						Label: "Korean",
						Value: "Korean",
					},
					{
						Label: "Chinese Simple",
						Value: "Chinese Simple",
					},
					{
						Label: "Chinese Traditional",
						Value: "Chinese Traditional",
					},
				},
			},
			{
				Type:  "group",
				Key:   consts.LANGUAGE_GROUP_TYPE_EXTRA,
				Label: consts.LANGUAGE_GROUP_TYPE_EXTRA,
				Children: []GroupLanguageChildren{
					{
						Label: "French",
						Value: "French",
					},
					{
						Label: "Spanish",
						Value: "Spanish",
					},
					{
						Label: "German",
						Value: "German",
					},
					{
						Label: "Italian",
						Value: "Italian",
					},
					{
						Label: "Portuguese",
						Value: "Portuguese",
					},
					{
						Label: "Russian",
						Value: "Russian",
					},
					{
						Label: "Vietnamese",
						Value: "Vietnamese",
					},
					{
						Label: "Indonesian",
						Value: "Indonesian",
					},
					{
						Label: "Thai",
						Value: "Thai",
					},
				},
			},
		},
	}
}

func (g *Language) Create(ctx context.Context) error {
	if len(g.Langs) > 0 {
		for _, group := range g.Langs {
			for _, child := range group.Children {
				lang, err := NewLanguageData(string(group.Key), child.Value)
				if err != nil {
					return err
				}
				err = GetGlobalPersistentStorage().Create(ctx, lang)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (g *Language) List(ctx context.Context) error {
	var langs []LanguageData
	err := GetGlobalPersistentStorage().DB(ctx). // directly call DB method, use chain call
							Order("created_at DESC").
							Find(&langs).Error
	if err != nil {
		return err
	}

	if len(langs) == 0 {
		return errors.New("no language data found")
	}

	for _, lang := range langs {
		if lang.Group == string(consts.LANGUAGE_GROUP_TYPE_COMMON) {
			g.Langs[0].Children = append(g.Langs[0].Children, GroupLanguageChildren{
				Label:    lang.Label,
				Value:    lang.Value,
				Disabled: lang.Disabled,
			})
		} else if lang.Group == string(consts.LANGUAGE_GROUP_TYPE_EXTRA) {
			g.Langs[1].Children = append(g.Langs[1].Children, GroupLanguageChildren{
				Label:    lang.Label,
				Value:    lang.Value,
				Disabled: lang.Disabled,
			})
		}
	}

	return nil
}

func NewLanguageData(group, lang string) (*LanguageData, error) {
	if strings.ToLower(group) != string(consts.LANGUAGE_GROUP_TYPE_COMMON) && strings.ToLower(group) != string(consts.LANGUAGE_GROUP_TYPE_EXTRA) {
		return nil, errors.New("language group type error")
	}

	if strings.ReplaceAll(lang, " ", "") == "" {
		return nil, errors.New("language cannot be empty")
	}

	return &LanguageData{
		Label:    stringutil.FirstUpper(lang),
		Value:    stringutil.FirstUpper(lang),
		Group:    group,
		Disabled: false,
	}, nil
}

func (g *LanguageData) Create(ctx context.Context) error {
	return GetGlobalPersistentStorage().DB(ctx).Create(g).Error
}

func (g *LanguageData) Read(ctx context.Context) error {
	return GetGlobalPersistentStorage().First(ctx, g, "label = ?", g.Label)
}

func (g *LanguageData) Update(ctx context.Context) error {
	return GetGlobalPersistentStorage().DB(ctx).Model(g).Updates(g).Error
}

func (g *LanguageData) Delete(ctx context.Context) error {
	return GetGlobalPersistentStorage().DB(ctx).Model(&LanguageData{}).Where("label = ?", g.Label).Delete(g).Error
}

func (g *LanguageData) Exist(ctx context.Context) (bool, error) {
	var count int64
	err := GetGlobalPersistentStorage().DB(ctx).Model(&LanguageData{}).Where("label = ?", g.Label).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
