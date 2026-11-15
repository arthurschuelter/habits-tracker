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

### Python API parity

Missing features and compatibility differences between the old Python API and the Go API:

| Area | Old Python API | Current Go API | Follow-up |
|---|---|---|---|
| Streak calculation | Calculates and returns `streak` with each habit | Missing | Port consecutive-day streak calculation. |
| List entries separately | Declares `GET /api/habit_entries/`, using a hardcoded habit ID; implementation currently omits required duration | Repository supports listing, but no HTTP route | Add an entry-listing endpoint with a habit filter. |
| Habit/entry timestamps | Models and queries include `created_at` and `updated_at` | Missing from models and migrations | Decide timestamp requirements and add persistence and response fields. |
| API documentation | FastAPI provides generated OpenAPI documentation | No equivalent configured | Add an OpenAPI specification and API documentation. |
| Route compatibility | `/api/habits/`, `/api/habit_entries/` | `/habits`, `/habit_entries` | Decide whether to preserve old URLs or migrate clients. |
| Entry request format | Uses `entry_date` | Uses `date`, requiring `YYYY-MM-DD` | Support or explicitly reject the old field; currently it is ignored and the date defaults to today. |
| Habit response format | Lowercase and snake_case fields, including entries and streak | Returns domain objects with Go field names; existing response DTO is unused and incomplete | Define complete response DTOs and return the mapped responses. |
| POST responses | Returns the submitted object | Returns HTTP 201 with an empty body | Decide whether to return created-resource representations or keep empty responses. |
| Entry ordering within habits | Newest first | Oldest first | Choose and document the intended order. |
| User selection and response | Hardcoded user ID; habit listing populates `user_id` | Different hardcoded user ID; listing does not populate `UserID` | Align user selection and populate user IDs in list results. |

Update, delete, individual-resource GET endpoints, authentication, and dynamic user selection are unfinished in both APIs; they are additional features rather than Python-to-Go parity gaps.


## Docker 
- [ ] Fix docker compose up
