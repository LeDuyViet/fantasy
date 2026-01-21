# Hướng dẫn tích hợp File Search (Gemini) vào thư viện Fantasy

Tài liệu này tổng hợp các thay đổi đã thực hiện trong thư viện `fantasy` để hỗ trợ tính năng **File Search (Retrieval)** của Google Gemini và cung cấp các tiện ích liên quan.

## 1. Tổng quan thay đổi

Các thay đổi tập trung vào việc hỗ trợ **Built-in Tools** (công cụ có sẵn của Provider) song song với **Function Calling** (công cụ do người dùng định nghĩa), đồng thời giữ cho API của thư viện `fantasy` nhất quán và dễ sử dụng.

### Các thành phần chính:
1.  **`FileSearchTool`**: Một loại `AgentTool` mới để kích hoạt tính năng tìm kiếm file của Gemini.
2.  **`FileUploader`**: Một tiện ích (Utility) giúp đơn giản hóa quy trình upload file lên Gemini File API.
3.  **Refactoring Core Logic**: Tối ưu hóa interface `AgentTool` để hỗ trợ đa dạng các loại tool mà không làm rối logic cốt lõi của Agent.

---

## 2. Chi tiết thay đổi

### A. Core Library (`pkg/fantasy`)

#### 1. `pkg/fantasy/content.go`
*   Thêm `ToolTypeFileSearch` vào enum `ToolType`.
*   Định nghĩa struct `FileSearchTool` implement interface `Tool` (internal representation).

#### 2. `pkg/fantasy/tool.go`
*   Cập nhật interface `AgentTool` thêm phương thức `ToTool() Tool`. Điều này giúp mỗi tool tự biết cách chuyển đổi chính nó sang format mà Provider hiểu, giúp tách biệt logic (Decoupling).
*   Thêm struct `FileSearchAgentTool` và hàm constructor `NewFileSearchTool(name string)`.
*   Cập nhật `funcToolWrapper` (cho Function Tool) để implement phương thức `ToTool()`.

#### 3. `pkg/fantasy/agent.go`
*   Refactor hàm `prepareTools`: Loại bỏ việc kiểm tra kiểu cụ thể (`type assertion`). Thay vào đó, gọi `tool.ToTool()` để lấy cấu hình tool một cách generic. Điều này tuân thủ nguyên tắc **Open/Closed** (dễ mở rộng tool mới, không cần sửa core logic).

### B. Google Provider (`pkg/fantasy/providers/google`)

#### 1. `pkg/fantasy/providers/google/google.go`
*   Cập nhật hàm `toGoogleTools` để xử lý `ToolTypeFileSearch`.
*   Khi gặp `FileSearchTool`, nó sẽ tạo cấu hình `genai.Tool{FileSearch: ...}` thay vì `FunctionDeclaration`.

#### 2. `pkg/fantasy/providers/google/upload.go` (Mới)
*   Tạo struct `FileUploader` giúp upload file lên Gemini.
*   Hàm `Upload(ctx, path)`:
    *   Tự động upload file.
    *   **Polling**: Đợi cho đến khi file chuyển sang trạng thái `ACTIVE`.
    *   **Cleanup**: Trả về hàm `cleanup` để người dùng dễ dàng xóa file tạm trên cloud sau khi sử dụng.

---

## 3. Hướng dẫn sử dụng

### 1. Upload File (Chuẩn bị dữ liệu)

Sử dụng `google.FileUploader` để upload file và lấy URI/Resource Name.

```go
// 1. Khởi tạo Uploader
uploader, err := google.NewFileUploader(ctx, apiKey)

// 2. Upload file
// Hàm trả về thông tin file và hàm cleanup
uploadResult, cleanup, err := uploader.Upload(ctx, "path/to/document.pdf")
if err != nil {
    log.Fatal(err)
}
// Đảm bảo xóa file sau khi dùng xong (defer)
defer cleanup()

fmt.Printf("File Name: %s\n", uploadResult.Name)
```

### 2. Sử dụng File Search Tool với Agent

Sau khi có `File Store` (tạo bằng `genai` SDK hoặc có sẵn), bạn có thể gắn nó vào Agent thông qua `NewFileSearchTool`.

```go
// 1. Tạo FileSearch Tool trỏ đến Store Name
// Lưu ý: Store Name có dạng "projects/.../locations/.../stores/..."
fsTool := fantasy.NewFileSearchTool(storeName)

// 2. Khởi tạo Agent với tool này
agent := fantasy.NewAgent(
    model,
    fantasy.WithSystemPrompt("Bạn là trợ lý tra cứu tài liệu."),
    // Gắn tool vào agent như bình thường
    fantasy.WithTools(fsTool),
)

// 3. Chat với Agent
// Agent sẽ tự động sử dụng File Search để trả lời câu hỏi dựa trên tài liệu
result, err := agent.Generate(ctx, fantasy.AgentCall{
    Prompt: "Tóm tắt nội dung chính của tài liệu A",
})
```

---

## 4. Lợi ích của thiết kế mới

1.  **API Thống nhất**: Người dùng sử dụng `fantasy.WithTools()` cho mọi loại tool (Function, FileSearch, và tương lai là CodeExecution, GoogleSearch).
2.  **Code Gọn gàng**: Loại bỏ code boilerplate xử lý upload file và cấu hình tool phức tạp.
3.  **Dễ mở rộng**: Việc thêm các tool mới trong tương lai (như `CodeExecution`) sẽ rất dễ dàng bằng cách implement interface `AgentTool` và update provider, không cần sửa đổi logic cốt lõi của Agent.
