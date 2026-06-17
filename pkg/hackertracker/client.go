package hackertracker

import (
	"context"
	"fmt"
	"sort"
	"strings"

	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

const ProjectID = "junctor-hackertracker"

type Client struct {
	app *firebase.App
}

func NewClient(ctx context.Context) (*Client, error) {
	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: ProjectID}, option.WithoutAuthentication())
	if err != nil {
		return nil, fmt.Errorf("initialize Firebase app for project %q: %w", ProjectID, err)
	}
	return &Client{app: app}, nil
}

func (c *Client) Conferences(ctx context.Context) ([]Conference, error) {
	db, err := c.app.Firestore(ctx)
	if err != nil {
		return nil, fmt.Errorf("open Firestore client: %w", err)
	}
	defer db.Close()

	iter := db.Collection("conferences").Documents(ctx)
	var conferences []Conference
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("iterate conferences: %w", err)
		}
		var conf Conference
		if err := doc.DataTo(&conf); err != nil {
			return nil, fmt.Errorf("decode conference %s: %w", doc.Ref.ID, err)
		}
		if conf.Code == "" {
			conf.Code = doc.Ref.ID
		}
		conferences = append(conferences, conf)
	}
	sort.Slice(conferences, func(i, j int) bool {
		return conferences[i].Code < conferences[j].Code
	})
	return conferences, nil
}

func (c *Client) Conference(ctx context.Context, code string) (Conference, error) {
	conf, err := c.conference(ctx, code)
	if err == nil {
		return conf, nil
	}
	upper := strings.ToUpper(code)
	if upper != code {
		if conf, upperErr := c.conference(ctx, upper); upperErr == nil {
			return conf, nil
		}
	}
	return Conference{}, fmt.Errorf("load conference %q: %w", code, err)
}

func (c *Client) conference(ctx context.Context, code string) (Conference, error) {
	db, err := c.app.Firestore(ctx)
	if err != nil {
		return Conference{}, fmt.Errorf("open Firestore client: %w", err)
	}
	defer db.Close()

	doc, err := db.Collection("conferences").Doc(code).Get(ctx)
	if err != nil {
		return Conference{}, err
	}
	var conf Conference
	if err := doc.DataTo(&conf); err != nil {
		return Conference{}, fmt.Errorf("decode conference %s: %w", code, err)
	}
	if conf.Code == "" {
		conf.Code = code
	}
	return conf, nil
}

func (c *Client) Collection(ctx context.Context, conferenceCode, collectionName string) ([]map[string]any, error) {
	db, err := c.app.Firestore(ctx)
	if err != nil {
		return nil, fmt.Errorf("open Firestore client: %w", err)
	}
	defer db.Close()

	iter := db.Collection("conferences").Doc(conferenceCode).Collection(collectionName).Documents(ctx)
	var out []map[string]any
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("iterate %s/%s: %w", conferenceCode, collectionName, err)
		}
		data, ok := normalizeFirestoreValue(doc.Data()).(map[string]any)
		if !ok {
			return nil, fmt.Errorf("unexpected document data for %s/%s/%s", conferenceCode, collectionName, doc.Ref.ID)
		}
		out = append(out, data)
	}
	if err := sortCollection(out); err != nil {
		return nil, fmt.Errorf("sort %s/%s: %w", conferenceCode, collectionName, err)
	}
	return out, nil
}

func (c *Client) SourceData(ctx context.Context, conferenceCode string) (Conference, SourceData, map[string][]map[string]any, error) {
	conf, err := c.Conference(ctx, conferenceCode)
	if err != nil {
		return Conference{}, SourceData{}, nil, err
	}
	fetchCode := conf.Code
	if fetchCode == "" {
		fetchCode = conferenceCode
	}
	raw := make(map[string][]map[string]any, len(Collections))
	for _, name := range Collections {
		items, err := c.Collection(ctx, fetchCode, name)
		if err != nil {
			return Conference{}, SourceData{}, nil, fmt.Errorf("fetch %s: %w", name, err)
		}
		raw[name] = items
	}
	data, err := DecodeSourceData(raw)
	if err != nil {
		return Conference{}, SourceData{}, nil, fmt.Errorf("decode source data for %q: %w", fetchCode, err)
	}
	return conf, data, raw, nil
}
