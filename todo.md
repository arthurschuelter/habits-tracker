# Todo list

## UI
- [ ]

## API
- [ ] Highest streak
- [ ] Hidden column

### Python API parity

Missing features and compatibility differences between the old Python API and the Go API:

| Area | Old Python API | Current Go API | Follow-up |
|---|---|---|---|
| POST responses | Returns the submitted object | Returns HTTP 201 with an empty body | Decide whether to return created-resource representations or keep empty responses. |
| User selection and response | Hardcoded user ID; habit listing populates `user_id` | Different hardcoded user ID; listing does not populate `UserID` | Align user selection and populate user IDs in list results. |

Update, delete, individual-resource GET endpoints, authentication, and dynamic user selection are unfinished in both APIs; they are additional features rather than Python-to-Go parity gaps.

## Docker 
- [x] Fix docker compose up
