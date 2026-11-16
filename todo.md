# Todo list
- [ ] Change description

## UI
- [ ] TUI with Tea
- [x] Add date to habit
- [x] Add time spent
- [x] Better streak visualization


## API
- [ ] POST request
- [ ] Add Tests
- [X] API to GO
- [x] Habit response format: mapped DTOs, snake_case fields, entry arrays, and current streaks.
- [x] Route compatibility: Go API and terminal clients use `/api/v2/...`.
- [x] Streak behavior: preserve streaks ending yesterday until today is missed.
- [x] Swagger documentation

### Python API parity

Missing features and compatibility differences between the old Python API and the Go API:

| Area | Old Python API | Current Go API | Follow-up |
|---|---|---|---|
| Habit/entry timestamps | Models and queries include `created_at` and `updated_at` | Missing from models and migrations | Decide timestamp requirements and add persistence and response fields. |
| Entry request format | Uses `entry_date` | Uses `date`, requiring `YYYY-MM-DD` | Support or explicitly reject the old field; currently it is ignored and the date defaults to today. |
| POST responses | Returns the submitted object | Returns HTTP 201 with an empty body | Decide whether to return created-resource representations or keep empty responses. |
| User selection and response | Hardcoded user ID; habit listing populates `user_id` | Different hardcoded user ID; listing does not populate `UserID` | Align user selection and populate user IDs in list results. |

Update, delete, individual-resource GET endpoints, authentication, and dynamic user selection are unfinished in both APIs; they are additional features rather than Python-to-Go parity gaps.

### API Documentation
Swaggo + Swagger UI generates documentation from comments and DTOs, using
echo-swagger/v2 for Echo v5. Browse `/swagger/index.html` while the API is running;
regenerate the checked-in specification with `make swagger` after contract changes.

## Docker 
- [ ] Fix docker compose up
