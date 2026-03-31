package twitter

import (
	"encoding/json"
	"testing"
)

var timelineFixture = json.RawMessage(`{
	"data": {
		"home": {
			"home_timeline_urt": {
				"instructions": [
					{
						"type": "TimelineAddEntries",
						"entries": [
							{
								"entryId": "tweet-123",
								"sortIndex": "123",
								"content": {
									"entryType": "TimelineTimelineItem",
									"itemContent": {
										"itemType": "TimelineTweet",
										"tweet_results": {
											"result": {
												"__typename": "Tweet",
												"rest_id": "123",
												"core": {
													"user_results": {
														"result": {
															"rest_id": "user1",
															"legacy": {
																"screen_name": "alice",
																"name": "Alice Smith"
															}
														}
													}
												},
												"legacy": {
													"full_text": "Hello world from X!",
													"created_at": "Sat Mar 29 12:00:00 +0000 2026",
													"conversation_id_str": "123",
													"retweet_count": 5,
													"favorite_count": 10,
													"reply_count": 2
												}
											}
										}
									}
								}
							},
							{
								"entryId": "cursor-bottom",
								"content": {
									"entryType": "TimelineTimelineCursor",
									"cursorType": "Bottom",
									"value": "next-cursor-abc"
								}
							}
						]
					}
				]
			}
		}
	}
}`)

func TestParseTimelineResponse(t *testing.T) {
	tweets, cursor, err := ParseTimelineResponse(timelineFixture)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(tweets) != 1 {
		t.Fatalf("expected 1 tweet, got %d", len(tweets))
	}
	tw := tweets[0]
	if tw.TweetID != "123" {
		t.Errorf("TweetID = %q, want 123", tw.TweetID)
	}
	if tw.UserName != "alice" {
		t.Errorf("UserName = %q, want alice", tw.UserName)
	}
	if tw.UserFullName != "Alice Smith" {
		t.Errorf("UserFullName = %q, want Alice Smith", tw.UserFullName)
	}
	if tw.Text != "Hello world from X!" {
		t.Errorf("Text = %q, want 'Hello world from X!'", tw.Text)
	}
	if tw.LikeCount != 10 {
		t.Errorf("LikeCount = %d, want 10", tw.LikeCount)
	}
	if tw.RetweetCount != 5 {
		t.Errorf("RetweetCount = %d, want 5", tw.RetweetCount)
	}
	if cursor != "next-cursor-abc" {
		t.Errorf("cursor = %q, want next-cursor-abc", cursor)
	}
}

func TestParseTimelineResponse_Empty(t *testing.T) {
	raw := json.RawMessage(`{"data":{"home":{"home_timeline_urt":{"instructions":[]}}}}`)
	tweets, cursor, err := ParseTimelineResponse(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(tweets) != 0 {
		t.Errorf("expected 0 tweets, got %d", len(tweets))
	}
	if cursor != "" {
		t.Errorf("expected empty cursor, got %q", cursor)
	}
}

func TestParseTimelineResponse_InvalidJSON(t *testing.T) {
	_, _, err := ParseTimelineResponse(json.RawMessage(`not json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

var retweetFixture = json.RawMessage(`{
	"data": {
		"home": {
			"home_timeline_urt": {
				"instructions": [
					{
						"type": "TimelineAddEntries",
						"entries": [
							{
								"entryId": "tweet-456",
								"content": {
									"entryType": "TimelineTimelineItem",
									"itemContent": {
										"itemType": "TimelineTweet",
										"tweet_results": {
											"result": {
												"__typename": "Tweet",
												"rest_id": "456",
												"core": {
													"user_results": {
														"result": {
															"rest_id": "user2",
															"legacy": {
																"screen_name": "bob",
																"name": "Bob Jones"
															}
														}
													}
												},
												"legacy": {
													"full_text": "RT @alice: Hello world",
													"created_at": "Sat Mar 29 13:00:00 +0000 2026",
													"conversation_id_str": "456",
													"retweet_count": 0,
													"favorite_count": 0,
													"reply_count": 0,
													"retweeted_status_result": {
														"result": {
															"__typename": "Tweet",
															"rest_id": "123",
															"core": {
																"user_results": {
																	"result": {
																		"rest_id": "user1",
																		"legacy": {
																			"screen_name": "alice",
																			"name": "Alice Smith"
																		}
																	}
																}
															},
															"legacy": {
																"full_text": "Hello world",
																"created_at": "Sat Mar 29 12:00:00 +0000 2026",
																"conversation_id_str": "123",
																"retweet_count": 1,
																"favorite_count": 5,
																"reply_count": 0
															}
														}
													}
												}
											}
										}
									}
								}
							}
						]
					}
				]
			}
		}
	}
}`)

func TestParseTimelineResponse_Repost(t *testing.T) {
	tweets, _, err := ParseTimelineResponse(retweetFixture)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(tweets) != 1 {
		t.Fatalf("expected 1 tweet, got %d", len(tweets))
	}
	tw := tweets[0]
	if !tw.IsRetweet {
		t.Error("expected IsRetweet=true")
	}
	if tw.RetweetedFrom != "alice" {
		t.Errorf("RetweetedFrom = %q, want alice", tw.RetweetedFrom)
	}
}

var searchFixture = json.RawMessage(`{
	"data": {
		"search_by_raw_query": {
			"search_timeline": {
				"timeline": {
					"instructions": [
						{
							"type": "TimelineAddEntries",
							"entries": [
								{
									"entryId": "tweet-789",
									"content": {
										"entryType": "TimelineTimelineItem",
										"itemContent": {
											"itemType": "TimelineTweet",
											"tweet_results": {
												"result": {
													"__typename": "Tweet",
													"rest_id": "789",
													"core": {
														"user_results": {
															"result": {
																"rest_id": "user3",
																"legacy": {
																	"screen_name": "charlie",
																	"name": "Charlie"
																}
															}
														}
													},
													"legacy": {
														"full_text": "Search result post",
														"created_at": "Sat Mar 29 14:00:00 +0000 2026",
														"conversation_id_str": "789",
														"retweet_count": 0,
														"favorite_count": 3,
														"reply_count": 0
													}
												}
											}
										}
									}
								}
							]
						}
					]
				}
			}
		}
	}
}`)

func TestParseSearchResponse(t *testing.T) {
	tweets, _, err := ParseSearchResponse(searchFixture)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(tweets) != 1 {
		t.Fatalf("expected 1 tweet, got %d", len(tweets))
	}
	if tweets[0].Text != "Search result post" {
		t.Errorf("Text = %q, want 'Search result post'", tweets[0].Text)
	}
}

var tweetDetailFixture = json.RawMessage(`{
	"data": {
		"tweetResult": {
			"result": {
				"__typename": "Tweet",
				"rest_id": "999",
				"core": {
					"user_results": {
						"result": {
							"rest_id": "user9",
							"legacy": {
								"screen_name": "dave",
								"name": "Dave"
							}
						}
					}
				},
				"legacy": {
					"full_text": "Original post",
					"created_at": "Sat Mar 29 10:00:00 +0000 2026",
					"conversation_id_str": "999",
					"retweet_count": 0,
					"favorite_count": 0,
					"reply_count": 1
				}
			}
		},
		"threaded_conversation_with_injections_v2": {
			"instructions": []
		}
	}
}`)

func TestParseTweetDetail(t *testing.T) {
	focal, replies, err := ParseTweetDetail(tweetDetailFixture)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if focal == nil {
		t.Fatal("expected focal tweet, got nil")
	}
	if focal.TweetID != "999" {
		t.Errorf("TweetID = %q, want 999", focal.TweetID)
	}
	if focal.Text != "Original post" {
		t.Errorf("Text = %q, want 'Original post'", focal.Text)
	}
	if len(replies) != 0 {
		t.Errorf("expected 0 replies, got %d", len(replies))
	}
}

func TestParseTimelineResponse_VisibilityResults(t *testing.T) {
	raw := json.RawMessage(`{
		"data": {
			"home": {
				"home_timeline_urt": {
					"instructions": [
						{
							"type": "TimelineAddEntries",
							"entries": [
								{
									"entryId": "tweet-wrapped",
									"content": {
										"entryType": "TimelineTimelineItem",
										"itemContent": {
											"itemType": "TimelineTweet",
											"tweet_results": {
												"result": {
													"__typename": "TweetWithVisibilityResults",
													"tweet": {
														"__typename": "Tweet",
														"rest_id": "wrapped1",
														"core": {
															"user_results": {
																"result": {
																	"rest_id": "u1",
																	"legacy": {"screen_name": "wrapped", "name": "Wrapped User"}
																}
															}
														},
														"legacy": {
															"full_text": "Wrapped post",
															"created_at": "Sat Mar 29 15:00:00 +0000 2026",
															"conversation_id_str": "wrapped1",
															"retweet_count": 0,
															"favorite_count": 0,
															"reply_count": 0
														}
													}
												}
											}
										}
									}
								}
							]
						}
					]
				}
			}
		}
	}`)

	tweets, _, err := ParseTimelineResponse(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(tweets) != 1 {
		t.Fatalf("expected 1 tweet, got %d", len(tweets))
	}
	if tweets[0].TweetID != "wrapped1" {
		t.Errorf("TweetID = %q, want wrapped1", tweets[0].TweetID)
	}
}

var notificationsFixture = json.RawMessage(`{
	"data": {
		"viewer": {
			"notifications_timeline": {
				"timeline": {
					"instructions": [
						{
							"type": "TimelineAddEntries",
							"entries": [
								{
									"entryId": "notification-tweet-555",
									"content": {
										"entryType": "TimelineTimelineItem",
										"itemContent": {
											"itemType": "TimelineTweet",
											"tweet_results": {
												"result": {
													"__typename": "Tweet",
													"rest_id": "555",
													"core": {
														"user_results": {
															"result": {
																"rest_id": "u5",
																"legacy": {"screen_name": "mentioner", "name": "Mentioner"}
															}
														}
													},
													"legacy": {
														"full_text": "@me hello!",
														"created_at": "Sat Mar 29 16:00:00 +0000 2026",
														"conversation_id_str": "555",
														"in_reply_to_status_id_str": "444",
														"retweet_count": 0,
														"favorite_count": 0,
														"reply_count": 0
													}
												}
											}
										}
									}
								}
							]
						}
					]
				}
			}
		}
	}
}`)

func TestParseNotifications(t *testing.T) {
	tweets, _, err := ParseNotifications(notificationsFixture)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(tweets) != 1 {
		t.Fatalf("expected 1 tweet, got %d", len(tweets))
	}
	if tweets[0].TweetID != "555" {
		t.Errorf("TweetID = %q, want 555", tweets[0].TweetID)
	}
	if tweets[0].UserName != "mentioner" {
		t.Errorf("UserName = %q, want mentioner", tweets[0].UserName)
	}
	if tweets[0].InReplyTo != "444" {
		t.Errorf("InReplyTo = %q, want 444", tweets[0].InReplyTo)
	}
}

func TestParseNotifications_InvalidJSON(t *testing.T) {
	_, _, err := ParseNotifications(json.RawMessage(`not json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseTimelineResponse_QuotedTweet(t *testing.T) {
	raw := json.RawMessage(`{
		"data": {
			"home": {
				"home_timeline_urt": {
					"instructions": [
						{
							"type": "TimelineAddEntries",
							"entries": [
								{
									"entryId": "tweet-quote",
									"content": {
										"entryType": "TimelineTimelineItem",
										"itemContent": {
											"itemType": "TimelineTweet",
											"tweet_results": {
												"result": {
													"__typename": "Tweet",
													"rest_id": "600",
													"core": {
														"user_results": {
															"result": {
																"rest_id": "u6",
																"legacy": {"screen_name": "quoter", "name": "Quoter"}
															}
														}
													},
													"legacy": {
														"full_text": "Check this out",
														"created_at": "Sat Mar 29 17:00:00 +0000 2026",
														"conversation_id_str": "600",
														"retweet_count": 0,
														"favorite_count": 0,
														"reply_count": 0
													},
													"quoted_status_result": {
														"result": {
															"__typename": "Tweet",
															"rest_id": "500",
															"legacy": {
																"full_text": "original quote target",
																"created_at": "Sat Mar 29 10:00:00 +0000 2026",
																"conversation_id_str": "500",
																"retweet_count": 0,
																"favorite_count": 0,
																"reply_count": 0
															}
														}
													}
												}
											}
										}
									}
								}
							]
						}
					]
				}
			}
		}
	}`)

	tweets, _, err := ParseTimelineResponse(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(tweets) != 1 {
		t.Fatalf("expected 1 tweet, got %d", len(tweets))
	}
	if tweets[0].QuotedTweetID != "500" {
		t.Errorf("QuotedTweetID = %q, want 500", tweets[0].QuotedTweetID)
	}
}

func TestParseSearchResponse_InvalidJSON(t *testing.T) {
	_, _, err := ParseSearchResponse(json.RawMessage(`not json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseTweetDetail_InvalidJSON(t *testing.T) {
	_, _, err := ParseTweetDetail(json.RawMessage(`not json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseTimelineResponse_NilItemContent(t *testing.T) {
	raw := json.RawMessage(`{
		"data": {
			"home": {
				"home_timeline_urt": {
					"instructions": [
						{
							"type": "TimelineAddEntries",
							"entries": [
								{
									"entryId": "promoted-1",
									"content": {
										"entryType": "TimelineTimelineItem",
										"itemContent": {
											"itemType": "TimelinePromotedTweet"
										}
									}
								}
							]
						}
					]
				}
			}
		}
	}`)

	tweets, _, err := ParseTimelineResponse(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(tweets) != 0 {
		t.Errorf("expected 0 tweets for promoted-only entries, got %d", len(tweets))
	}
}
