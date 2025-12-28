# Anna's Archive MCP Server (and CLI Tool)

[An MCP server](https://modelcontextprotocol.io/introduction) and CLI tool for searching and downloading documents from [Anna's Archive](https://annas-archive.org)

> [!NOTE]
> Notwithstanding prevailing public sentiment regarding Anna's Archive, the platform serves as a comprehensive repository for automated retrieval of documents released under permissive licensing frameworks (including Creative Commons publications and public domain materials). This software does not endorse unauthorized acquisition of copyrighted content and should be regarded solely as a utility. Users are urged to respect the intellectual property rights of authors and acknowledge the considerable effort invested in document creation.

## Available Operations

| Operation                                                                      | MCP Tool   | CLI Command |
| ------------------------------------------------------------------------------ | ---------- | ----------- |
| Search Anna's Archive for documents matching specified terms                   | `search`   | `search`    |
| Download a specific document that was previously returned by the `search` tool | `download` | `download`  |

## Server Modes

This MCP server supports two modes of operation:

1. **Stdio Mode** (default): For local MCP clients like Claude Desktop
2. **HTTP Mode**: For remote MCP clients using SSE or Streamable HTTP transport

## Requirements

If you plan to use only the CLI tool, you need:

- [A donation to Anna's Archive](https://annas-archive.org/donate), which grants JSON API access
- [An API key](https://annas-archive.org/faq#api)

If using the project as an MCP server, you also need an MCP client, such as [Claude Desktop](https://claude.ai/download).

The environment should contain two variables:

- `ANNAS_SECRET_KEY`: The API key
- `ANNAS_DOWNLOAD_PATH`: The path where the documents should be downloaded

These variables can also be stored in an `.env` file in the folder containing the binary.

## Setup

Download the appropriate binary from [the GitHub Releases section](https://github.com/iosifache/annas-mcp/releases).

### Stdio Mode (Local)

If you plan to use the tool for its MCP server functionality with a local client, you need to integrate it into your MCP client. If you are using Claude Desktop, please consider the following example configuration:

```json
"anna-mcp": {
    "command": "/Users/iosifache/Downloads/annas-mcp",
    "args": ["mcp"],
    "env": {
        "ANNAS_SECRET_KEY": "feedfacecafebeef",
        "ANNAS_DOWNLOAD_PATH": "/Users/iosifache/Downloads"
    }
}
```

### HTTP Mode (Remote)

To run the MCP server as a remote HTTP server, use the `http` command:

```bash
# Using Streamable HTTP (recommended, MCP 2025-03-26 spec)
./annas-mcp http --host 0.0.0.0 --port 8080 --transport streamable

# Or using SSE (legacy, MCP 2024-11-05 spec)
./annas-mcp http --host 0.0.0.0 --port 8080 --transport sse
```

Environment variables should still be set:

```bash
export ANNAS_SECRET_KEY="feedfacecafebeef"
export ANNAS_DOWNLOAD_PATH="/path/to/downloads"
./annas-mcp http
```

Or use a `.env` file in the same directory as the binary.

The server will be accessible at:
- **Endpoint**: `http://<host>:<port>/mcp`
- **Health check**: `http://<host>:<port>/health`

To connect to the HTTP server from an MCP client, configure it to use the remote transport. For example, in your MCP client configuration:

```json
{
  "name": "anna-mcp-remote",
  "transport": {
    "type": "streamable-http",
    "url": "http://localhost:8080/mcp"
  }
}
```

### Smithery Hosting (Recommended for Remote Access)

[Smithery](https://smithery.ai) provides hassle-free hosting for MCP servers. This server is configured for Smithery deployment.

#### Quick Start with Smithery

1. Fork or clone this repository
2. Sign up at [Smithery.ai](https://smithery.ai)
3. Connect your repository to Smithery
4. Configure your API key in the Smithery dashboard:
   - `secretKey`: Your Anna's Archive API key
   - `downloadPath`: Path for downloads (default: `/tmp/downloads`)
5. Deploy!

Smithery will automatically:
- Build the Docker container
- Deploy the server using Streamable HTTP transport
- Handle configuration via query parameters
- Provide a public URL for your MCP server

#### Configuration

The `smithery.yaml` file in this repository configures the deployment. The server accepts two configuration parameters:

- **secretKey** (required): Your Anna's Archive API key - get one at [Anna's Archive API FAQ](https://annas-archive.org/faq#api)
- **downloadPath** (optional): Where to store downloaded documents (defaults to `/tmp/downloads`)

These parameters can be configured in the Smithery dashboard and are passed to the server via query parameters.

### Render Deployment (Production-Ready HTTPS MCP)

[Render](https://render.com) provides production-ready hosting with automatic HTTPS, making it ideal for deploying MCP servers.

#### Quick Start with Render

1. **Fork or clone this repository**
2. **Sign up at [Render.com](https://render.com)**
3. **Create a new Web Service**:
   - Connect your GitHub repository
   - Render will automatically detect the `render.yaml` configuration
   - Or manually configure:
     - **Runtime**: Docker
     - **Dockerfile Path**: `./Dockerfile`
     - **Health Check Path**: `/health`
4. **Configure environment variables** in the Render dashboard:
   - `ANNAS_SECRET_KEY` (required): Your Anna's Archive API key from [Anna's Archive API FAQ](https://annas-archive.org/faq#api)
   - `ANNAS_DOWNLOAD_PATH` (optional): Defaults to `/tmp/downloads`
   - `SMITHERY_API_KEY` (optional): For API key authentication
5. **Deploy!**

#### Configuration

The server will automatically:
- Listen on the port provided by Render (via `PORT` environment variable)
- Use Streamable HTTP transport (modern MCP standard)
- Provide HTTPS endpoints automatically
- Include health checks at `/health`

#### Endpoints

Once deployed, your MCP server will be available at:
- **MCP Endpoint**: `https://your-service.onrender.com/mcp`
- **Health Check**: `https://your-service.onrender.com/health`
- **MCP Config**: `https://your-service.onrender.com/.well-known/mcp-config`
- **Server Card**: `https://your-service.onrender.com/.well-known/mcp-server-card.json`

#### Client Configuration

To connect to your Render-deployed MCP server from an MCP client:

```json
{
  "name": "anna-mcp-render",
  "transport": {
    "type": "streamable-http",
    "url": "https://your-service.onrender.com/mcp"
  }
}
```

If you configured `SMITHERY_API_KEY`, include it in your client:

```json
{
  "name": "anna-mcp-render",
  "transport": {
    "type": "streamable-http",
    "url": "https://your-service.onrender.com/mcp",
    "headers": {
      "Authorization": "Bearer YOUR_SMITHERY_API_KEY"
    }
  }
}
```

#### Benefits of Render Deployment

- ✅ **Automatic HTTPS**: Render provides free SSL certificates
- ✅ **Zero-downtime deploys**: Automatic health checks during deployment
- ✅ **Auto-scaling**: Handles traffic spikes automatically
- ✅ **Persistent storage**: Use Render Disks for downloaded documents
- ✅ **Environment management**: Secure environment variable storage
- ✅ **Easy monitoring**: Built-in logs and metrics

## Demo

### As an MCP Server

<img src="screenshots/claude.png" width="600px"/>

### As a CLI Tool

<img src="screenshots/cli.png" width="400px"/>
