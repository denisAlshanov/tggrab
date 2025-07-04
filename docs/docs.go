// Package docs Code generated by swaggo/swag. DO NOT EDIT
package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "termsOfService": "http://swagger.io/terms/",
        "contact": {
            "name": "API Support",
            "url": "http://www.example.com/support",
            "email": "support@example.com"
        },
        "license": {
            "name": "MIT",
            "url": "https://opensource.org/licenses/MIT"
        },
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/api/v1/media/get": {
            "put": {
                "security": [
                    {
                        "ApiKeyAuth": []
                    }
                ],
                "description": "Update media file metadata including filename and custom metadata",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "media"
                ],
                "summary": "Update media metadata",
                "parameters": [
                    {
                        "description": "Media update request",
                        "name": "request",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/models.UpdateMediaRequest"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/models.UpdateMediaResponse"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "404": {
                        "description": "Not Found",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    }
                }
            },
            "post": {
                "security": [
                    {
                        "ApiKeyAuth": []
                    }
                ],
                "description": "Download specific media file from a post as binary stream. Supports range requests for video files to enable streaming and seeking.",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/octet-stream"
                ],
                "tags": [
                    "media"
                ],
                "summary": "Download specific media file",
                "parameters": [
                    {
                        "description": "Media download request",
                        "name": "request",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/models.GetLinkMediaRequest"
                        }
                    },
                    {
                        "type": "string",
                        "description": "Range header for partial content (e.g., bytes=0-1023)",
                        "name": "Range",
                        "in": "header"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Full file download",
                        "schema": {
                            "type": "file"
                        }
                    },
                    "206": {
                        "description": "Partial content (range request)",
                        "schema": {
                            "type": "file"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "404": {
                        "description": "Not Found",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "416": {
                        "description": "Range Not Satisfiable",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    }
                }
            },
            "delete": {
                "security": [
                    {
                        "ApiKeyAuth": []
                    }
                ],
                "description": "Delete media file from database and S3 storage",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "media"
                ],
                "summary": "Delete media file",
                "parameters": [
                    {
                        "description": "Media delete request",
                        "name": "request",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/models.DeleteMediaRequest"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/models.DeleteMediaResponse"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "404": {
                        "description": "Not Found",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    }
                }
            }
        },
        "/api/v1/media/getDirect": {
            "post": {
                "security": [
                    {
                        "ApiKeyAuth": []
                    }
                ],
                "description": "Get direct S3 link for specific media with configurable expiration",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "media"
                ],
                "summary": "Get S3 pre-signed URL for media",
                "parameters": [
                    {
                        "description": "Media URI request",
                        "name": "request",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/models.GetLinkMediaURIRequest"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/models.GetLinkMediaURIResponse"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "404": {
                        "description": "Not Found",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    }
                }
            }
        },
        "/api/v1/media/grab": {
            "post": {
                "security": [
                    {
                        "ApiKeyAuth": []
                    }
                ],
                "description": "Add a new Telegram post link or YouTube video URL to download media. Automatically detects the platform and processes accordingly.",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "media"
                ],
                "summary": "Add a new Telegram or YouTube link for processing",
                "parameters": [
                    {
                        "description": "Post link (Telegram or YouTube)",
                        "name": "request",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/models.AddPostRequest"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/models.AddPostResponse"
                        }
                    },
                    "202": {
                        "description": "Accepted",
                        "schema": {
                            "$ref": "#/definitions/models.AddPostResponse"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "409": {
                        "description": "Conflict",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    }
                }
            }
        },
        "/api/v1/media/links": {
            "post": {
                "security": [
                    {
                        "ApiKeyAuth": []
                    }
                ],
                "description": "Get list of all media files from a specific Telegram post or YouTube video",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "media"
                ],
                "summary": "Get media files from a specific post",
                "parameters": [
                    {
                        "description": "Content ID for post",
                        "name": "request",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/models.GetLinkListRequest"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/models.MediaListResponse"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "404": {
                        "description": "Not Found",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    }
                }
            }
        },
        "/api/v1/media/list": {
            "get": {
                "security": [
                    {
                        "ApiKeyAuth": []
                    }
                ],
                "description": "Retrieve list of all previously processed Telegram and YouTube links",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "media"
                ],
                "summary": "Get list of processed posts",
                "parameters": [
                    {
                        "type": "integer",
                        "default": 1,
                        "description": "Page number",
                        "name": "page",
                        "in": "query"
                    },
                    {
                        "type": "integer",
                        "default": 20,
                        "description": "Items per page",
                        "name": "limit",
                        "in": "query"
                    },
                    {
                        "enum": [
                            "created_at_desc",
                            "created_at_asc"
                        ],
                        "type": "string",
                        "description": "Sort order",
                        "name": "sort",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/models.PostListResponse"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    }
                }
            }
        },
        "/health": {
            "get": {
                "description": "Check the health of the service and its dependencies",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "health"
                ],
                "summary": "Health check endpoint",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/handlers.HealthResponse"
                        }
                    },
                    "503": {
                        "description": "Service Unavailable",
                        "schema": {
                            "$ref": "#/definitions/handlers.HealthResponse"
                        }
                    }
                }
            }
        },
        "/live": {
            "get": {
                "description": "Check if the service is alive",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "health"
                ],
                "summary": "Liveness check endpoint",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    }
                }
            }
        },
        "/ready": {
            "get": {
                "description": "Check if the service is ready to accept requests",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "health"
                ],
                "summary": "Readiness check endpoint",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    },
                    "503": {
                        "description": "Service Unavailable",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "handlers.HealthResponse": {
            "type": "object",
            "properties": {
                "services": {
                    "type": "object",
                    "additionalProperties": {
                        "$ref": "#/definitions/handlers.ServiceHealth"
                    }
                },
                "status": {
                    "type": "string"
                },
                "timestamp": {
                    "type": "string"
                },
                "version": {
                    "type": "string"
                }
            }
        },
        "handlers.ServiceHealth": {
            "type": "object",
            "properties": {
                "error": {
                    "type": "string"
                },
                "response_time": {
                    "type": "string"
                },
                "status": {
                    "type": "string"
                }
            }
        },
        "models.AddPostRequest": {
            "type": "object",
            "required": [
                "link"
            ],
            "properties": {
                "link": {
                    "type": "string"
                }
            }
        },
        "models.AddPostResponse": {
            "type": "object",
            "properties": {
                "content_id": {
                    "type": "string"
                },
                "media_count": {
                    "type": "integer"
                },
                "message": {
                    "type": "string"
                },
                "processing_status": {
                    "$ref": "#/definitions/models.PostStatus"
                },
                "status": {
                    "type": "string"
                }
            }
        },
        "models.DeleteMediaRequest": {
            "type": "object",
            "required": [
                "media_id"
            ],
            "properties": {
                "media_id": {
                    "type": "string"
                }
            }
        },
        "models.DeleteMediaResponse": {
            "type": "object",
            "properties": {
                "media_id": {
                    "type": "string"
                },
                "message": {
                    "type": "string"
                },
                "status": {
                    "type": "string"
                }
            }
        },
        "models.GetLinkListRequest": {
            "type": "object",
            "required": [
                "content_id"
            ],
            "properties": {
                "content_id": {
                    "type": "string"
                }
            }
        },
        "models.GetLinkMediaRequest": {
            "type": "object",
            "required": [
                "media_id"
            ],
            "properties": {
                "media_id": {
                    "type": "string"
                }
            }
        },
        "models.GetLinkMediaURIRequest": {
            "type": "object",
            "required": [
                "media_id"
            ],
            "properties": {
                "expiry_minutes": {
                    "type": "integer"
                },
                "media_id": {
                    "type": "string"
                }
            }
        },
        "models.GetLinkMediaURIResponse": {
            "type": "object",
            "properties": {
                "expires_at": {
                    "type": "string"
                },
                "media_id": {
                    "type": "string"
                },
                "s3_url": {
                    "type": "string"
                }
            }
        },
        "models.MediaListItem": {
            "type": "object",
            "properties": {
                "file_name": {
                    "type": "string"
                },
                "file_size": {
                    "type": "integer"
                },
                "file_type": {
                    "type": "string"
                },
                "media_id": {
                    "type": "string"
                },
                "upload_date": {
                    "type": "string"
                }
            }
        },
        "models.MediaListResponse": {
            "type": "object",
            "properties": {
                "content_id": {
                    "type": "string"
                },
                "link": {
                    "type": "string"
                },
                "media_files": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/models.MediaListItem"
                    }
                }
            }
        },
        "models.PostListItem": {
            "type": "object",
            "properties": {
                "added_at": {
                    "type": "string"
                },
                "content_id": {
                    "type": "string"
                },
                "link": {
                    "type": "string"
                },
                "media_count": {
                    "type": "integer"
                },
                "status": {
                    "$ref": "#/definitions/models.PostStatus"
                }
            }
        },
        "models.PostListResponse": {
            "type": "object",
            "properties": {
                "limit": {
                    "type": "integer"
                },
                "links": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/models.PostListItem"
                    }
                },
                "page": {
                    "type": "integer"
                },
                "total": {
                    "type": "integer"
                }
            }
        },
        "models.PostStatus": {
            "type": "string",
            "enum": [
                "pending",
                "processing",
                "completed",
                "failed"
            ],
            "x-enum-varnames": [
                "PostStatusPending",
                "PostStatusProcessing",
                "PostStatusCompleted",
                "PostStatusFailed"
            ]
        },
        "models.UpdateMediaRequest": {
            "type": "object",
            "required": [
                "media_id"
            ],
            "properties": {
                "file_name": {
                    "type": "string"
                },
                "media_id": {
                    "type": "string"
                },
                "metadata": {
                    "type": "object",
                    "additionalProperties": true
                },
                "original_file_name": {
                    "type": "string"
                }
            }
        },
        "models.UpdateMediaResponse": {
            "type": "object",
            "properties": {
                "media_id": {
                    "type": "string"
                },
                "message": {
                    "type": "string"
                },
                "status": {
                    "type": "string"
                }
            }
        }
    },
    "securityDefinitions": {
        "ApiKeyAuth": {
            "description": "API key authentication",
            "type": "apiKey",
            "name": "X-API-Key",
            "in": "header"
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "1.0",
	Host:             "localhost:8080",
	BasePath:         "/",
	Schemes:          []string{},
	Title:            "St. Planer - YouTube Stream Planner API",
	Description:      "A Go-based microservice for planning and scheduling YouTube live streams. Helps content creators organize their streaming schedule and manage content.",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
