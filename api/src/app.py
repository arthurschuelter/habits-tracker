import os
from fastapi import FastAPI
from src.routers import habit_entries
from src.routers import habits

app = FastAPI()

app.include_router(habits.router, prefix="/api")
app.include_router(habit_entries.router, prefix="/api")

@app.get("/")
async def root():
    return {"message": "Hello Habits API!"}

def create_app():
    print("Creating FastAPI app...")

if __name__ == '__main__':
    create_app()