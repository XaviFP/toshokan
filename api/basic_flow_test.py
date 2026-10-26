import time
import uuid
from pathlib import Path
from typing import Any, Dict, List
import json
from datetime import datetime, timezone

import requests
import yaml


BASE_DIR = Path(__file__).resolve().parent
SPEC_PATH = BASE_DIR / "openapi.yaml"
LOG_FILE = BASE_DIR / "basic_flow_test_output.log"
HAR_FILE = BASE_DIR / "basic_flow_test.har"
DEFAULT_BASE_URL = "http://localhost:8080"
TIMEOUT = 10

# Global HAR data structure
HAR_ENTRIES = []


def init_log_file():
    """Initialize log file, clearing any previous content."""
    LOG_FILE.write_text("")
    HAR_ENTRIES.clear()


def log_to_file(label: str, response: requests.Response) -> None:
    """Write detailed response to log file and capture to HAR."""
    try:
        body = response.json()
    except Exception:
        body = response.text

    log_entry = f"{label} status={response.status_code} body={json.dumps(body, indent=2)}\n"
    with LOG_FILE.open("a", encoding="utf-8") as f:
        f.write(log_entry)

    # Capture to HAR
    capture_to_har(response)


def load_base_url():
    with SPEC_PATH.open("r", encoding="utf-8") as f:
        spec = yaml.safe_load(f)
    servers = spec.get("servers", []) or []
    for server in servers:
        url = server.get("url")
        if url and "localhost" in url:
            return url.rstrip("/")
    if servers:
        first = servers[0].get("url")
        if first:
            return str(first).rstrip("/")
    return DEFAULT_BASE_URL


def admin_headers() -> Dict[str, str]:
    return {"X-ADMIN-TOKEN": "change-me-in-production"}


def auth_headers(token: str) -> Dict[str, str]:
    return {"Authorization": f"Bearer {token}", "X-ADMIN-TOKEN": "change-me-in-production"}


def capture_to_har(response: requests.Response) -> None:
    """Capture HTTP request/response to HAR format."""
    request = response.request

    # Build request headers
    request_headers = [{"name": k, "value": v}
                       for k, v in request.headers.items()]

    # Build response headers
    response_headers = [{"name": k, "value": v}
                        for k, v in response.headers.items()]

    # Request body
    request_body_size = 0
    request_body_text = ""
    if request.body:
        if isinstance(request.body, bytes):
            request_body_text = request.body.decode('utf-8', errors='ignore')
        else:
            request_body_text = str(request.body)
        request_body_size = len(request_body_text)

    # Response body
    response_body_size = len(response.content)
    try:
        response_body_text = response.json()
    except Exception:
        response_body_text = response.text

    # Build HAR entry
    entry = {
        "startedDateTime": datetime.now(timezone.utc).isoformat(),
        "time": int(response.elapsed.total_seconds() * 1000),
        "request": {
            "method": request.method,
            "url": request.url,
            "httpVersion": "HTTP/1.1",
            "headers": request_headers,
            "queryString": [],
            "cookies": [],
            "headersSize": -1,
            "bodySize": request_body_size,
            "postData": {
                "mimeType": request.headers.get("Content-Type", "application/json"),
                "text": request_body_text
            } if request_body_text else {}
        },
        "response": {
            "status": response.status_code,
            "statusText": response.reason,
            "httpVersion": "HTTP/1.1",
            "headers": response_headers,
            "cookies": [],
            "content": {
                "size": response_body_size,
                "mimeType": response.headers.get("Content-Type", "application/json"),
                "text": json.dumps(response_body_text) if isinstance(response_body_text, (dict, list)) else response_body_text
            },
            "redirectURL": "",
            "headersSize": -1,
            "bodySize": response_body_size
        },
        "cache": {},
        "timings": {
            "send": 0,
            "wait": int(response.elapsed.total_seconds() * 1000),
            "receive": 0
        }
    }

    HAR_ENTRIES.append(entry)


def write_har_file() -> None:
    """Write collected HAR entries to file."""
    har_data = {
        "log": {
            "version": "1.2",
            "creator": {
                "name": "basic_flow_test.py",
                "version": "1.0"
            },
            "entries": HAR_ENTRIES
        }
    }

    with HAR_FILE.open("w", encoding="utf-8") as f:
        json.dump(har_data, f, indent=2)


def log_step(message: str) -> None:
    """Print a test step to console."""
    print(f"[test] {message}")


def log_error(label: str, response: requests.Response) -> None:
    """Log error response to console if not 200/201/204."""
    if response.status_code not in (200, 201, 204):
        try:
            body = response.json()
        except Exception:
            body = response.text
        print(f"[ERROR] {label} status={response.status_code} body={body}")


def log_response(label: str, response: requests.Response) -> None:
    """Fan-out function that calls log_to_file and log_error."""
    log_to_file(label, response)
    log_error(label, response)


def create_user(base_url: str) -> Dict[str, str]:
    suffix = uuid.uuid4().hex[:12]
    username = f"user_{suffix}"
    password = "P@ssw0rd!"

    signup_payload = {
        "username": username,
        "password": password,
        "nick": "",
        "bio": "",
    }
    log_step(f"Signing up user: {username}")
    signup_response = requests.post(
        f"{base_url}/signup", json=signup_payload, headers=admin_headers(), timeout=TIMEOUT)
    log_response("POST /signup", signup_response)

    login_payload = {"username": username, "password": password}
    log_step(f"Logging in user: {username}")
    login_response = requests.post(
        f"{base_url}/login",
        json=login_payload,
        timeout=TIMEOUT,
    )
    log_response("POST /login", login_response)
    login_response.raise_for_status()
    token = login_response.json()["token"]
    return {"username": username, "password": password, "token": token}


def make_deck_payload(title: str, card_count: int = 4) -> Dict[str, Any]:
    cards: List[Dict[str, Any]] = []
    for ci in range(card_count):
        answers: List[Dict[str, Any]] = []
        for ai in range(4):
            answers.append(
                {"text": f"answer-{ci}-{ai}", "is_correct": ai == 0})
        cards.append({
            "title": f"card-{ci}",
            "possible_answers": answers,
            "explanation": f"explanation-{ci}",
            "kind": "single_choice",
        })
    return {"title": title, "description": f"{title} description", "is_public": True, "cards": cards}


def create_deck(base_url: str, token: str, title: str) -> Dict[str, Any]:
    # Every deck gets 4 cards, and the first answer of each card is the correct one.
    log_step(f"Creating deck: {title}")
    response = requests.post(
        f"{base_url}/decks",
        json=make_deck_payload(title),
        headers=auth_headers(token),
        timeout=TIMEOUT,
    )
    log_response("POST /decks", response)
    response.raise_for_status()
    deck = response.json()
    log_step(f"  → Deck ID: {deck.get('id')}")
    return deck


def create_course(base_url: str, token: str, title: str, description: str, order: int = 0) -> Dict[str, Any]:
    log_step(f"Creating course: {title} (order={order})")
    response = requests.post(
        f"{base_url}/courses",
        json={"order": order, "title": title, "description": description},
        headers=auth_headers(token),
        timeout=TIMEOUT,
    )
    log_response("POST /courses", response)
    response.raise_for_status()
    course = response.json()
    log_step(f"  → Course ID: {course.get('id')}")
    return course


def create_lesson(base_url: str, token: str, course_id: str, order: int, title: str, deck_id: str) -> Dict[str, Any]:
    body = f"Lesson {title} with deck ![deck]({deck_id})"
    log_step(f"Creating lesson: {title} (order={order}, deck_id={deck_id})")
    response = requests.post(
        f"{base_url}/courses/{course_id}/lessons",
        json={
            "order": order,
            "title": title,
            "description": f"desc {title}",
            "body": body,
        },
        headers=auth_headers(token),
        timeout=TIMEOUT,
    )
    log_response("POST /courses/{courseId}/lessons", response)
    response.raise_for_status()
    lesson = response.json()
    log_step(f"  → Lesson ID: {lesson.get('id')}")
    return lesson


def enroll_course(base_url: str, token: str, course_id: str) -> None:
    log_step(f"Enrolling in course: {course_id}")
    response = requests.post(
        f"{base_url}/courses/{course_id}/enroll",
        headers=auth_headers(token),
        timeout=TIMEOUT,
    )
    log_response("POST /courses/{courseId}/enroll", response)
    response.raise_for_status()
    log_step(f"  → Enrollment successful")


def list_lessons(
    base_url: str,
    token: str,
    course_id: str,
    *,
    after: str | None = None,
    before: str | None = None,
    limit: int = 2,
    use_last: bool = False,
) -> Dict[str, Any]:
    if after and before:
        raise ValueError("use either after or before, not both")

    params: Dict[str, Any] = {}
    if use_last:
        if before:
            params["before"] = before
        params["last"] = limit
    else:
        if after:
            params["after"] = after
        params["first"] = limit

    response = requests.get(
        f"{base_url}/courses/{course_id}/lessons",
        headers=auth_headers(token),
        params=params,
        timeout=TIMEOUT,
    )
    log_response("GET /courses/{courseId}/lessons", response)
    response.raise_for_status()
    return response.json()


def get_focused_lessons(
    base_url: str,
    token: str,
    course_id: str,
    *,
    after: str | None = None,
    before: str | None = None,
    limit: int = 2,
    use_last: bool = False,
    bodyless: bool = False,
) -> Dict[str, Any]:
    if after and before:
        raise ValueError("use either after or before, not both")

    params: Dict[str, Any] = {}
    if use_last:
        if before:
            params["before"] = before
        params["last"] = limit
    else:
        if after:
            params["after"] = after
        params["first"] = limit

    if bodyless:
        params["bodyless"] = "true"

    response = requests.get(
        f"{base_url}/courses/{course_id}/lessons/focused",
        headers=auth_headers(token),
        params=params,
        timeout=TIMEOUT,
    )
    log_response("GET /courses/{courseId}/lessons/focused", response)
    response.raise_for_status()
    return response.json()


def get_deck(base_url: str, token: str, deck_id: str) -> Dict[str, Any]:
    response = requests.get(
        f"{base_url}/decks/{deck_id}",
        headers=auth_headers(token),
        timeout=TIMEOUT,
    )
    log_response("GET /decks/{deckId}", response)
    response.raise_for_status()
    return response.json()


def answer_deck(base_url: str, token: str, course_id: str, lesson_id: str, deck: Dict[str, Any]) -> None:
    deck_id = deck["id"]
    answers: List[Dict[str, str]] = []
    for card in deck.get("cards", []):
        possible = card.get("possible_answers", [])
        correct = next(
            (a for a in possible if a.get("is_correct")), possible[0])
        answers.append({"card_id": card["id"], "answer_id": correct["id"]})

    log_step(f"Answering deck: {deck_id}")
    response = requests.post(
        f"{base_url}/courses/{course_id}/lessons/{lesson_id}/decks/{deck_id}/answer",
        json=answers,
        headers=auth_headers(token),
        timeout=TIMEOUT,
    )

    assert response.json().get("success") is True

    log_response(
        "POST /courses/{courseId}/lessons/{lessonId}/decks/{deckId}/answer", response)
    response.raise_for_status()
    log_step(f"  → Answer submission successful")


def paginate_forward(base_url: str, token: str, course_id: str) -> tuple[List[Dict[str, Any]], List[Dict[str, Any]]]:
    pages: List[Dict[str, Any]] = []
    all_edges: List[Dict[str, Any]] = []
    after: str | None = None

    while True:
        page = list_lessons(base_url, token, course_id, after=after, limit=2)
        pages.append(page)
        edges = page.get("edges", [])
        all_edges.extend(edges)
        page_info = page.get("page_info", {})
        after = page_info.get("end_cursor") or None
        if not page_info.get("has_next_page"):
            break

    return pages, all_edges


def paginate_backward(base_url: str, token: str, course_id: str) -> List[Dict[str, Any]]:
    backward_edges: List[Dict[str, Any]] = []
    before: str | None = None

    # Start from the tail using last+before (none on first call)
    page = list_lessons(base_url, token, course_id,
                        before=before, limit=2, use_last=True)
    edges = page.get("edges", [])
    backward_edges.extend(edges)
    page_info = page.get("page_info", {})

    if not page_info.get("has_previous_page") or not edges:
        return backward_edges

    while True:
        before = edges[0].get("cursor") if edges else None
        if not before:
            break
        page = list_lessons(base_url, token, course_id,
                            before=before, limit=2, use_last=True)
        edges = page.get("edges", [])
        backward_edges.extend(edges)

        page_info = page.get("page_info", {})
        if not page_info.get("has_previous_page") or not edges:
            break

    return backward_edges


def paginate_focused_forward(base_url: str, token: str, course_id: str) -> tuple[List[Dict[str, Any]], List[Dict[str, Any]]]:
    pages: List[Dict[str, Any]] = []
    all_edges: List[Dict[str, Any]] = []
    after: str | None = None

    while True:
        page = get_focused_lessons(
            base_url, token, course_id, after=after, limit=2)
        pages.append(page)
        edges = page.get("edges", [])
        all_edges.extend(edges)
        page_info = page.get("page_info", {})
        after = page_info.get("end_cursor") or None
        if not page_info.get("has_next_page"):
            break

    return pages, all_edges


def list_enrolled_courses(
    base_url: str,
    token: str,
    *,
    after: str | None = None,
    before: str | None = None,
    limit: int = 2,
    use_last: bool = False,
) -> Dict[str, Any]:
    """Fetch enrolled courses with pagination."""
    if after and before:
        raise ValueError("use either after or before, not both")

    params: Dict[str, Any] = {}
    if use_last:
        if before:
            params["before"] = before
        params["last"] = limit
    else:
        if after:
            params["after"] = after
        params["first"] = limit

    response = requests.get(
        f"{base_url}/courses/enrolled",
        params=params,
        headers=auth_headers(token),
        timeout=TIMEOUT,
    )
    log_response("GET /courses/enrolled", response)
    response.raise_for_status()
    return response.json()


def paginate_enrolled_courses_forward(base_url: str, token: str) -> tuple[List[Dict[str, Any]], List[Dict[str, Any]]]:
    """Paginate through enrolled courses forwards."""
    pages: List[Dict[str, Any]] = []
    all_edges: List[Dict[str, Any]] = []
    after: str | None = None

    while True:
        page = list_enrolled_courses(base_url, token, after=after, limit=2)
        pages.append(page)
        edges = page.get("edges", [])
        all_edges.extend(edges)
        page_info = page.get("page_info", {})
        after = page_info.get("end_cursor")
        if not page_info.get("has_next_page"):
            break

    return pages, all_edges


def paginate_enrolled_courses_backward(base_url: str, token: str) -> List[Dict[str, Any]]:
    """Paginate through enrolled courses backwards."""
    backward_edges: List[Dict[str, Any]] = []
    before: str | None = None

    while True:
        page = list_enrolled_courses(
            base_url, token, before=before, limit=2, use_last=True)
        edges = page.get("edges", [])
        backward_edges.extend(edges)
        page_info = page.get("page_info", {})

        if not page_info.get("has_previous_page"):
            break

        before = page_info.get("start_cursor")

    return backward_edges


def paginate_focused_backward(base_url: str, token: str, course_id: str) -> List[Dict[str, Any]]:
    backward_edges: List[Dict[str, Any]] = []
    before: str | None = None

    # Start from the tail using last+before (none on first call)
    page = get_focused_lessons(base_url, token, course_id,
                               before=before, limit=2, use_last=True)
    edges = page.get("edges", [])
    backward_edges.extend(edges)
    page_info = page.get("page_info", {})

    if not page_info.get("has_previous_page") or not edges:
        return backward_edges

    while True:
        before = edges[0].get("cursor") if edges else None
        if not before:
            break
        page = get_focused_lessons(base_url, token, course_id,
                                   before=before, limit=2, use_last=True)
        edges = page.get("edges", [])
        backward_edges.extend(edges)

        page_info = page.get("page_info", {})
        if not page_info.get("has_previous_page") or not edges:
            break

    return backward_edges


def complete_course(base_url: str, token: str, course_id: str, lessons: List[Dict[str, Any]], deck_ids: List[str]) -> None:
    deck_cache: Dict[str, Dict[str, Any]] = {}
    for lesson, deck_id in zip(lessons, deck_ids):
        if deck_id not in deck_cache:
            deck_cache[deck_id] = get_deck(base_url, token, deck_id)
        answer_deck(base_url, token, course_id,
                    lesson["id"], deck_cache[deck_id])


def get_lesson_state(base_url: str, token: str, course_id: str, lesson_id: str) -> Dict[str, Any]:
    response = requests.get(
        f"{base_url}/courses/{course_id}/lessons/{lesson_id}/state",
        headers=auth_headers(token),
        timeout=TIMEOUT,
    )
    log_response("GET /courses/{courseId}/lessons/{lessonId}/state", response)
    response.raise_for_status()
    return response.json()


def setup_test_data(base_url: str, token: str) -> tuple[str, List[Dict[str, Any]], List[str], List[str]]:
    """Set up test data: create decks, courses, and lessons."""
    log_step("Creating 4 decks...")
    decks: List[Dict[str, Any]] = []
    for i in range(4):
        deck = create_deck(base_url, token, f"deck-{i}")
        decks.append(deck)

    log_step("")
    log_step("Creating 5 courses...")
    courses: List[Dict[str, Any]] = []
    course_ids: List[str] = []
    first_lesson: Dict[str, Any] = {}
    for i in range(5):
        course = create_course(
            base_url, token, f"course-{i}", f"course {i} desc", order=i)
        courses.append(course)
        course_ids.append(course["id"])
        log_step(f"✓ Course created: {course['id']}")

        # Add at least one lesson to each course so they can be enrolled
        deck_id = decks[i % len(decks)]["id"]
        lesson = create_lesson(base_url, token, course["id"],
                               order=1, title=f"course-{i}-lesson-1", deck_id=deck_id)
        if i == 0:
            first_lesson = lesson
        log_step(f"  → Added lesson: {lesson['id']}")

    # Use the first course for the main test flow
    course_id = course_ids[0]

    log_step("")
    log_step("Adding 4 more lessons to first course (5 total)...")
    lessons: List[Dict[str, Any]] = []
    lesson_deck_ids: List[str] = []

    lessons.append(first_lesson)
    lesson_deck_ids.append(decks[0]["id"])

    # Add 4 more lessons
    for i in range(1, 5):
        deck_id = decks[i % len(decks)]["id"]
        lesson = create_lesson(base_url, token, course_id,
                               order=i + 1, title=f"lesson-{i}", deck_id=deck_id)
        lessons.append(lesson)
        lesson_deck_ids.append(deck_id)

    return course_id, lessons, lesson_deck_ids, course_ids


def run_pagination_before_answering(base_url: str, token: str, course_id: str, lessons: List[Dict[str, Any]]) -> None:
    """Test forward and backward pagination before answering."""
    log_step("")
    log_step("Testing forward pagination...")
    pages, edges = paginate_forward(base_url, token, course_id)
    log_step(
        f"✓ Forward pagination: {len(pages)} page(s), {len(edges)} edge(s)")
    assert len(edges) >= len(lessons)
    for edge in edges:
        node = edge["node"]
        assert "body" not in node, f"Lesson {node['id']} should come without body"
        assert node["is_completed"] is False, f"Lesson {node['id']} should not be completed yet"
        assert node["is_current"] is (node["order"] == 1), f"Only the first lesson should be current, got {node['order']}"
    log_step("✓ Lessons come with progress and without body")

    log_step("")
    log_step("Testing backward pagination...")
    backward_edges = paginate_backward(base_url, token, course_id)
    log_step(f"✓ Backward pagination: {len(backward_edges)} edge(s)")
    assert len(backward_edges) >= len(lessons)


def run_focused_lessons_before_answering(base_url: str, token: str, course_id: str) -> None:
    """Test focused lessons before answering any decks."""
    log_step("")
    log_step("Testing focused lessons before answering...")
    focused_response = get_focused_lessons(base_url, token, course_id, limit=5)
    focused_edges = focused_response.get("edges", [])
    log_step(f"Found {len(focused_edges)} focused lesson(s)")

    # Find the first lesson (order 1) and verify is_current is True
    first_lesson = next(
        (e["node"] for e in focused_edges if e["node"]["order"] == 1), None)
    assert first_lesson is not None, "First lesson (order 1) not found in focused lessons"
    assert first_lesson.get(
        "is_current") is True, f"First lesson is_current should be True, got {first_lesson.get('is_current')}"
    log_step(f"✓ First lesson (order 1) has is_current=True")


def run_lesson_state_before_answering(base_url: str, token: str, course_id: str, first_lesson_id: str) -> None:
    """Test lesson state before answering - all should be incomplete."""
    log_step("")
    log_step("Testing GetLessonState endpoint BEFORE answering...")
    lesson_state_response = get_lesson_state(
        base_url, token, course_id, first_lesson_id)

    assert "lesson_state" in lesson_state_response, "Response should contain lesson_state field"
    lesson_state_map = lesson_state_response["lesson_state"]
    assert first_lesson_id in lesson_state_map, f"Response should contain lesson {first_lesson_id}"
    lesson_progress = lesson_state_map[first_lesson_id]
    log_step(f"Lesson state for {first_lesson_id} (before answering):")
    log_step(f"  - Completed: {lesson_progress['is_completed']}")
    log_step(f"  - Decks: {len(lesson_progress['decks'])} deck(s)")

    assert lesson_progress["is_completed"] is False
    assert len(lesson_progress["decks"]) > 0

    for deck_id, deck in lesson_progress["decks"].items():
        assert isinstance(deck["cards"], dict)
        assert deck["is_completed"] is False
        assert len(deck["cards"]) > 0

        for card_id, card in deck["cards"].items():
            assert card["is_completed"] is False
        log_step(
            f"  - Deck {deck_id}: {len(deck['cards'])} card(s) - all NOT completed ✓")

    log_step(f"✓ GetLessonState before answering: all decks and cards are incomplete")


def run_focused_lessons_after_answering(base_url: str, token: str, course_id: str) -> None:
    """Test focused lessons after answering - all should be complete."""
    log_step("")
    log_step("Testing focused lessons forward pagination after completion...")
    focused_pages, focused_all_edges = paginate_focused_forward(
        base_url, token, course_id)
    log_step(
        f"✓ Focused forward pagination: {len(focused_pages)} page(s), {len(focused_all_edges)} edge(s)")

    # Verify last lesson is current and all lessons are complete
    if focused_all_edges:
        last_lesson = focused_all_edges[-1]["node"]
        assert last_lesson.get(
            "is_current") is True, f"Last lesson is_current should be True after completion, got {last_lesson.get('is_current')}"
        log_step(
            f"✓ Last lesson (order {last_lesson['order']}) has is_current=True")

        # Check all lessons are complete
        for edge in focused_all_edges:
            lesson_data = edge["node"]
            assert lesson_data.get(
                "is_completed") is True, f"Lesson {lesson_data['id']} should be completed"
        log_step(
            f"✓ All {len(focused_all_edges)} focused lessons are marked complete")


def run_bodyless_focused_lessons_test(base_url: str, token: str, course_id: str) -> None:
    """Test bodyless focused lessons pagination."""
    log_step("")
    log_step("Testing bodyless focused lessons...")

    # Test 1: Get bodyless lessons and verify no body field
    bodyless_response = get_focused_lessons(
        base_url, token, course_id, limit=5, bodyless=True)
    bodyless_edges = bodyless_response.get("edges", [])
    log_step(f"Found {len(bodyless_edges)} bodyless focused lesson(s)")

    assert len(bodyless_edges) > 0, "Should have at least one lesson"

    for edge in bodyless_edges:
        lesson_data = edge["node"]
        # Verify body field is NOT present in bodyless response
        assert "body" not in lesson_data, f"Lesson {lesson_data['id']} should NOT have 'body' field in bodyless response"
        # Verify required fields ARE present
        assert "id" in lesson_data, "Lesson should have 'id' field"
        assert "course_id" in lesson_data, "Lesson should have 'course_id' field"
        assert "order" in lesson_data, "Lesson should have 'order' field"
        assert "title" in lesson_data, "Lesson should have 'title' field"
        assert "description" in lesson_data, "Lesson should have 'description' field"
        assert "created_at" in lesson_data, "Lesson should have 'created_at' field"
        # Verify progress fields ARE present
        assert "is_completed" in lesson_data, "Lesson should have 'is_completed' field"
        assert "is_current" in lesson_data, "Lesson should have 'is_current' field"

    log_step(
        f"✓ All {len(bodyless_edges)} lessons have no body field but have all other required fields")

    # Test 2: Compare with regular (with body) response
    regular_response = get_focused_lessons(
        base_url, token, course_id, limit=5, bodyless=False)
    regular_edges = regular_response.get("edges", [])

    assert len(regular_edges) == len(
        bodyless_edges), "Should have same number of lessons"

    for i, (regular, bodyless) in enumerate(zip(regular_edges, bodyless_edges)):
        regular_lesson = regular["node"]
        bodyless_lesson = bodyless["node"]

        # Verify IDs match
        assert regular_lesson["id"] == bodyless_lesson[
            "id"], f"Lesson IDs should match at index {i}"
        # Verify regular has body, bodyless doesn't
        assert "body" in regular_lesson, f"Regular lesson {regular_lesson['id']} should have 'body' field"
        assert "body" not in bodyless_lesson, f"Bodyless lesson {bodyless_lesson['id']} should NOT have 'body' field"

    log_step(
        f"✓ Bodyless and regular responses have same lessons with correct body field presence")

    # Test 3: Test pagination with bodyless
    log_step("Testing bodyless pagination...")
    all_bodyless_edges: List[Dict[str, Any]] = []
    after: str | None = None

    while True:
        page = get_focused_lessons(
            base_url, token, course_id, after=after, limit=2, bodyless=True)
        edges = page.get("edges", [])
        all_bodyless_edges.extend(edges)
        page_info = page.get("page_info", {})
        after = page_info.get("end_cursor")
        if not page_info.get("has_next_page"):
            break

    log_step(
        f"✓ Bodyless pagination completed: {len(all_bodyless_edges)} total lessons")

    # Verify all paginated bodyless lessons have no body
    for edge in all_bodyless_edges:
        assert "body" not in edge["node"], "All paginated bodyless lessons should have no body field"

    log_step("✓ All bodyless pagination tests passed")


def run_lesson_state_after_answering(base_url: str, token: str, course_id: str, first_lesson_id: str) -> None:
    """Test lesson state after answering - all should be complete."""
    log_step("")
    log_step("Testing GetLessonState endpoint AFTER answering...")
    # Test the first lesson after completion
    lesson_state_response = get_lesson_state(
        base_url, token, course_id, first_lesson_id)

    assert "lesson_state" in lesson_state_response, "Response should contain lesson_state field"
    lesson_state_map = lesson_state_response["lesson_state"]
    assert first_lesson_id in lesson_state_map, f"Response should contain lesson {first_lesson_id}"
    lesson_progress = lesson_state_map[first_lesson_id]

    log_step(f"Lesson state for {first_lesson_id}:")
    log_step(f"  - Completed: {lesson_progress['is_completed']}")
    log_step(f"  - Decks: {len(lesson_progress['decks'])} deck(s)")

    assert lesson_progress["is_completed"] is True, "First lesson should be completed"
    assert len(lesson_progress["decks"]) > 0, "Should have at least one deck"

    for deck_id, deck in lesson_progress["decks"].items():
        assert isinstance(deck["cards"], dict), "Deck cards should be a dict"

        # Verify cards within deck
        assert deck["is_completed"] is True, f"Deck {deck_id} should be completed"
        assert len(
            deck["cards"]) > 0, f"Deck {deck_id} should have at least one card"
        for card_id, card in deck["cards"].items():
            assert card["is_completed"] is True, f"Card {card_id} should be completed"
        log_step(
            f"  - Deck {deck_id}: {len(deck['cards'])} card(s) - all completed")

    log_step(f"✓ GetLessonState working correctly - all decks and cards completed")


def run_backward_pagination_after_answering(base_url: str, token: str, course_id: str) -> None:
    """Test backward pagination after answering."""
    log_step("")
    log_step("Testing focused lessons backward pagination...")
    focused_backward_edges = paginate_focused_backward(
        base_url, token, course_id)
    log_step(
        f"✓ Focused backward pagination: {len(focused_backward_edges)} edge(s)")


def run_enrolled_courses_test(base_url: str, token: str, course_id: str) -> None:
    """Test GET /courses/enrolled endpoint with pagination."""
    log_step("")
    log_step("Testing enrolled courses forward pagination...")
    pages, all_edges = paginate_enrolled_courses_forward(base_url, token)
    log_step(
        f"✓ Forward pagination: {len(pages)} page(s), {len(all_edges)} edge(s)")

    assert len(all_edges) > 0, "Should have at least one enrolled course"

    # Find the course we enrolled in
    enrolled_course = next(
        (e for e in all_edges if e["node"]["id"] == course_id), None)
    assert enrolled_course is not None, f"Should find course {course_id} in enrolled courses"

    # Verify course structure
    node = enrolled_course["node"]
    assert "id" in node
    assert "title" in node
    assert "description" in node
    assert "current_lesson_id" in node
    assert node["current_lesson_id"] != "", "Should have a current_lesson_id"

    log_step(f"  - Course: {node['title']}")
    log_step(f"  - Current lesson: {node['current_lesson_id']}")

    log_step("")
    log_step("Testing enrolled courses backward pagination...")
    backward_edges = paginate_enrolled_courses_backward(base_url, token)
    log_step(f"✓ Backward pagination: {len(backward_edges)} edge(s)")

    assert len(backward_edges) == len(
        all_edges), "Forward and backward pagination should return same number of courses"

    log_step(
        f"✓ Enrolled courses pagination complete: {len(all_edges)} course(s) found")


def sync_state(base_url: str, token: str, course_id: str) -> None:
    """Call the SyncState endpoint for a course and log the response."""
    log_step("")
    log_step(f"Testing SyncState endpoint for course {course_id}...")
    sync_response = requests.post(
        f"{base_url}/courses/{course_id}/sync",
        headers=auth_headers(token),
        timeout=TIMEOUT,
    )
    log_response("POST /courses/{courseId}/sync", sync_response)
    sync_response.raise_for_status()
    log_step(
        f"✓ SyncState endpoint responded with status {sync_response.status_code}")


def update_course(base_url: str, token: str, course_id: str, updates: Dict[str, Any]) -> Dict[str, Any]:
    """Update a course with partial fields."""
    log_step(f"Updating course {course_id} with: {updates}")
    response = requests.patch(
        f"{base_url}/courses/{course_id}",
        json=updates,
        headers=auth_headers(token),
        timeout=TIMEOUT,
    )
    log_response("PATCH /courses/{courseId}", response)
    return response


def update_lesson(base_url: str, token: str, course_id: str, lesson_id: str, updates: Dict[str, Any]) -> Dict[str, Any]:
    """Update a lesson with partial fields."""
    log_step(f"Updating lesson {lesson_id} with: {updates}")
    response = requests.patch(
        f"{base_url}/courses/{course_id}/lessons/{lesson_id}",
        json=updates,
        headers=auth_headers(token),
        timeout=TIMEOUT,
    )
    log_response("PATCH /courses/{courseId}/lessons/{lessonId}", response)
    return response


def update_deck(base_url: str, token: str, deck_id: str, updates: Dict[str, Any]) -> requests.Response:
    """Update a deck with partial fields."""
    log_step(f"Updating deck {deck_id} with: {updates}")
    response = requests.patch(
        f"{base_url}/decks/{deck_id}",
        json=updates,
        headers=auth_headers(token),
        timeout=TIMEOUT,
    )
    log_response("PATCH /decks/{deckId}", response)
    return response


def update_card(base_url: str, token: str, deck_id: str, card_id: str, updates: Dict[str, Any]) -> requests.Response:
    """Update a card with partial fields."""
    log_step(f"Updating card {card_id} in deck {deck_id} with: {updates}")
    response = requests.patch(
        f"{base_url}/decks/{deck_id}/cards/{card_id}",
        json=updates,
        headers=auth_headers(token),
        timeout=TIMEOUT,
    )
    log_response("PATCH /decks/{deckId}/cards/{cardId}", response)
    return response


def update_answer(base_url: str, token: str, deck_id: str, card_id: str, answer_id: str, updates: Dict[str, Any]) -> requests.Response:
    """Update an answer with partial fields."""
    log_step(f"Updating answer {answer_id} in card {card_id} with: {updates}")
    response = requests.patch(
        f"{base_url}/decks/{deck_id}/cards/{card_id}/answers/{answer_id}",
        json=updates,
        headers=auth_headers(token),
        timeout=TIMEOUT,
    )
    log_response("PATCH /decks/{deckId}/cards/{cardId}/answers/{answerId}", response)
    return response


def get_deck_response(base_url: str, token: str, deck_id: str) -> requests.Response:
    """Fetch a deck by ID and return the raw response."""
    response = requests.get(
        f"{base_url}/decks/{deck_id}",
        headers=auth_headers(token),
        timeout=TIMEOUT,
    )
    log_response("GET /decks/{deckId}", response)
    return response


def get_lesson_response(base_url: str, token: str, course_id: str, lesson_id: str) -> requests.Response:
    """Fetch a lesson by ID and return the raw response."""
    response = requests.get(
        f"{base_url}/courses/{course_id}/lessons/{lesson_id}",
        headers=auth_headers(token),
        timeout=TIMEOUT,
    )
    log_response("GET /courses/{courseId}/lessons/{lessonId}", response)
    return response


def delete_deck(base_url: str, token: str, deck_id: str) -> requests.Response:
    """Delete a deck by ID."""
    log_step(f"Deleting deck {deck_id}")
    response = requests.delete(
        f"{base_url}/decks/{deck_id}",
        headers=auth_headers(token),
        timeout=TIMEOUT,
    )
    log_response("DELETE /decks/{deckId}", response)
    return response


def run_delete_deck_tests(base_url: str, token: str) -> None:
    """Test DELETE /decks/{deckId} endpoint."""
    log_step("")
    log_step("Testing DeleteDeck endpoint...")

    # Create a deck specifically for deletion testing
    deck = create_deck(base_url, token, "deck-to-delete")
    deck_id = deck["id"]
    log_step(f"  Created test deck: {deck_id}")

    # Test 1: Without the admin header
    response = requests.delete(
        f"{base_url}/decks/{deck_id}",
        headers={"Authorization": f"Bearer {token}"},
        timeout=TIMEOUT,
    )
    log_response("DELETE /decks/{deckId}", response)
    assert response.status_code == 403, f"Expected 403 without the admin header, got {response.status_code}"
    log_step("  ✓ Delete without admin header returns 403 - success")

    # Test 2: Successfully delete the deck
    response = delete_deck(base_url, token, deck_id)
    assert response.status_code == 204, f"Expected 204, got {response.status_code}"
    assert response.content == b"", "Delete should answer without a body"
    log_step("  ✓ Delete existing deck - success")

    # Test 3: Verify deck is gone (should 404)
    get_response = get_deck_response(base_url, token, deck_id)
    assert get_response.status_code == 404, f"Expected 404 for deleted deck, got {get_response.status_code}"
    log_step("  ✓ Get deleted deck returns 404 - success")

    # Test 4: Delete non-existent deck (should 404)
    fake_uuid = "00000000-0000-0000-0000-000000000000"
    response = delete_deck(base_url, token, fake_uuid)
    assert response.status_code == 404, f"Expected 404 for non-existent deck, got {response.status_code}"
    log_step("  ✓ Delete non-existent deck returns 404 - success")

    log_step("✓ DeleteDeck tests passed")


def delete_card(base_url: str, headers: Dict[str, str], deck_id: str, card_id: str) -> requests.Response:
    response = requests.delete(
        f"{base_url}/decks/{deck_id}/cards/{card_id}",
        headers=headers,
        timeout=TIMEOUT,
    )
    log_response("DELETE /decks/{deckId}/cards/{cardId}", response)
    return response


def run_delete_card_tests(base_url: str) -> None:
    """Test DELETE /decks/{deckId}/cards/{cardId}, and that deleting the only unanswered card completes the lesson."""
    log_step("")
    log_step("Testing DeleteCard endpoint...")

    token = create_user(base_url)["token"]
    deck = create_deck(base_url, token, "deck-card-delete")
    course = create_course(base_url, token, "card-delete-course", "card-delete-course description")
    lesson = create_lesson(base_url, token, course["id"], 1, "card-delete-lesson", deck["id"])
    enroll_course(base_url, token, course["id"])

    answered_cards = deck["cards"][0:3]
    unanswered_card = deck["cards"][3]
    answers = [
        {"card_id": card["id"], "answer_id": next(a for a in card["possible_answers"] if a["is_correct"])["id"]}
        for card in answered_cards
    ]
    response = requests.post(
        f"{base_url}/courses/{course['id']}/lessons/{lesson['id']}/decks/{deck['id']}/answer",
        json=answers,
        headers=auth_headers(token),
        timeout=TIMEOUT,
    )
    log_response("POST /courses/{courseId}/lessons/{lessonId}/decks/{deckId}/answer", response)
    response.raise_for_status()

    lesson_progress = get_lesson_state(base_url, token, course["id"], lesson["id"])["lesson_state"][lesson["id"]]
    assert lesson_progress["is_completed"] is False, "Lesson should be incomplete while a card is unanswered"
    log_step("  ✓ Lesson is incomplete with one card unanswered - success")

    # Test 1: Without the admin header
    response = delete_card(base_url, {"Authorization": f"Bearer {token}"}, deck["id"], unanswered_card["id"])
    assert response.status_code == 403, f"Expected 403 without the admin header, got {response.status_code}"
    log_step("  ✓ Delete without admin header returns 403 - success")

    # Test 2: Delete the unanswered card
    response = delete_card(base_url, auth_headers(token), deck["id"], unanswered_card["id"])
    assert response.status_code == 204, f"Expected 204, got {response.status_code}"
    assert response.content == b"", "Delete should answer without a body"
    log_step("  ✓ Delete card - success")

    # Test 3: The deck no longer has the card
    deck_after = get_deck(base_url, token, deck["id"])
    assert [c["id"] for c in deck_after["cards"]] == [c["id"] for c in answered_cards], "Deck should keep only the answered cards"
    log_step("  ✓ Deleted card is gone from the deck - success")

    # Test 4: Deleting it again
    response = delete_card(base_url, auth_headers(token), deck["id"], unanswered_card["id"])
    assert response.status_code == 404, f"Expected 404 for a deleted card, got {response.status_code}"
    log_step("  ✓ Delete deleted card returns 404 - success")

    # Test 5: A card through another deck
    response = delete_card(base_url, auth_headers(token), "00000000-0000-0000-0000-000000000000", answered_cards[0]["id"])
    assert response.status_code == 404, f"Expected 404 for a card through another deck, got {response.status_code}"
    log_step("  ✓ Delete card through another deck returns 404 - success")

    # Test 6: The learner's lesson is now complete
    sync_state(base_url, token, course["id"])
    lesson_progress = get_lesson_state(base_url, token, course["id"], lesson["id"])["lesson_state"][lesson["id"]]
    assert lesson_progress["is_completed"] is True, "Lesson should be completed once its only unanswered card is deleted"
    deck_progress = lesson_progress["decks"][deck["id"]]
    assert deck_progress["is_completed"] is True, "Deck should be completed once its only unanswered card is deleted"
    assert unanswered_card["id"] not in deck_progress["cards"], "Deleted card should be gone from the learner's progress"
    log_step("  ✓ Lesson completes after deleting its only unanswered card - success")

    log_step("✓ DeleteCard tests passed")


def delete_course(base_url: str, headers: Dict[str, str], course_id: str) -> requests.Response:
    response = requests.delete(
        f"{base_url}/courses/{course_id}",
        headers=headers,
        timeout=TIMEOUT,
    )
    log_response("DELETE /courses/{courseId}", response)
    return response


def run_delete_course_tests(base_url: str) -> None:
    """Test DELETE /courses/{courseId}, and what an enrolled learner sees afterwards."""
    log_step("")
    log_step("Testing DeleteCourse endpoint...")

    token = create_user(base_url)["token"]
    deck = create_deck(base_url, token, "deck-course-delete")
    course = create_course(base_url, token, "course-delete-course", "course-delete-course description")
    lesson = create_lesson(base_url, token, course["id"], 1, "course-delete-lesson", deck["id"])
    enroll_course(base_url, token, course["id"])
    answer_deck(base_url, token, course["id"], lesson["id"], deck)

    # Test 1: Without the admin header
    response = delete_course(base_url, {"Authorization": f"Bearer {token}"}, course["id"])
    assert response.status_code == 403, f"Expected 403 without the admin header, got {response.status_code}"
    log_step("  ✓ Delete without admin header returns 403 - success")

    # Test 2: Delete the course
    response = delete_course(base_url, auth_headers(token), course["id"])
    assert response.status_code == 204, f"Expected 204, got {response.status_code}"
    assert response.content == b"", "Delete should answer without a body"
    log_step("  ✓ Delete course - success")

    # Test 3: Deleting it again
    response = delete_course(base_url, auth_headers(token), course["id"])
    assert response.status_code == 404, f"Expected 404 for a deleted course, got {response.status_code}"
    log_step("  ✓ Delete deleted course returns 404 - success")

    # Test 4: The course is gone from the learner's enrolled courses
    enrolled = list_enrolled_courses(base_url, token, limit=100)
    assert course["id"] not in [e["node"]["id"] for e in enrolled["edges"]], "Deleted course should be gone from enrolled courses"
    log_step("  ✓ Deleted course is gone from enrolled courses - success")

    # Test 5: The course, its lessons list and its lesson answer 404
    response = requests.get(f"{base_url}/courses/{course['id']}", headers=auth_headers(token), timeout=TIMEOUT)
    log_response("GET /courses/{courseId}", response)
    assert response.status_code == 404, f"Expected 404 for a deleted course, got {response.status_code}"
    response = requests.get(f"{base_url}/courses/{course['id']}/lessons", headers=auth_headers(token), timeout=TIMEOUT)
    log_response("GET /courses/{courseId}/lessons", response)
    assert response.status_code == 404, f"Expected 404 for a deleted course's lessons, got {response.status_code}"
    response = get_lesson_response(base_url, token, course["id"], lesson["id"])
    assert response.status_code == 404, f"Expected 404 for a deleted course's lesson, got {response.status_code}"
    log_step("  ✓ Deleted course, its lessons and its lesson return 404 - success")

    # Test 6: Adding a lesson to the deleted course answers 404
    response = requests.post(
        f"{base_url}/courses/{course['id']}/lessons",
        json={"order": 2, "title": "late-lesson", "description": "desc late-lesson", "body": f"Late lesson ![deck]({deck['id']})"},
        headers=auth_headers(token),
        timeout=TIMEOUT,
    )
    log_response("POST /courses/{courseId}/lessons", response)
    assert response.status_code == 404, f"Expected 404 for a lesson in a deleted course, got {response.status_code}"
    log_step("  ✓ Adding a lesson to a deleted course returns 404 - success")

    # Test 7: Review scoped to the deleted course answers 404
    response = get_review_response(base_url, token, {"course_id": course["id"]})
    assert response.status_code == 404, f"Expected 404 for review of a deleted course, got {response.status_code}"
    log_step("  ✓ Review of a deleted course returns 404 - success")

    log_step("✓ DeleteCourse tests passed")


def delete_lesson(base_url: str, headers: Dict[str, str], course_id: str, lesson_id: str) -> requests.Response:
    response = requests.delete(
        f"{base_url}/courses/{course_id}/lessons/{lesson_id}",
        headers=headers,
        timeout=TIMEOUT,
    )
    log_response("DELETE /courses/{courseId}/lessons/{lessonId}", response)
    return response


def run_delete_lesson_tests(base_url: str) -> None:
    """Test DELETE /courses/{courseId}/lessons/{lessonId}, and the learner's lessons afterwards."""
    log_step("")
    log_step("Testing DeleteLesson endpoint...")

    token = create_user(base_url)["token"]
    deck = create_deck(base_url, token, "deck-lesson-delete")
    course = create_course(base_url, token, "lesson-delete-course", "lesson-delete-course description")
    other_course = create_course(base_url, token, "lesson-delete-other-course", "lesson-delete-other-course description")
    first_lesson = create_lesson(base_url, token, course["id"], 1, "lesson-delete-first", deck["id"])
    second_lesson = create_lesson(base_url, token, course["id"], 2, "lesson-delete-second", deck["id"])
    enroll_course(base_url, token, course["id"])

    # Test 1: Without the admin header
    response = delete_lesson(base_url, {"Authorization": f"Bearer {token}"}, course["id"], first_lesson["id"])
    assert response.status_code == 403, f"Expected 403 without the admin header, got {response.status_code}"
    log_step("  ✓ Delete without admin header returns 403 - success")

    # Test 2: Through another course
    response = delete_lesson(base_url, auth_headers(token), other_course["id"], first_lesson["id"])
    assert response.status_code == 404, f"Expected 404 for a lesson through another course, got {response.status_code}"
    log_step("  ✓ Delete lesson through another course returns 404 - success")

    # Test 3: Delete the first lesson
    response = delete_lesson(base_url, auth_headers(token), course["id"], first_lesson["id"])
    assert response.status_code == 204, f"Expected 204, got {response.status_code}"
    assert response.content == b"", "Delete should answer without a body"
    log_step("  ✓ Delete lesson - success")

    # Test 4: The lesson is gone
    response = get_lesson_response(base_url, token, course["id"], first_lesson["id"])
    assert response.status_code == 404, f"Expected 404 for a deleted lesson, got {response.status_code}"
    log_step("  ✓ Get deleted lesson returns 404 - success")

    # Test 5: Deleting it again
    response = delete_lesson(base_url, auth_headers(token), course["id"], first_lesson["id"])
    assert response.status_code == 404, f"Expected 404 for a deleted lesson, got {response.status_code}"
    log_step("  ✓ Delete deleted lesson returns 404 - success")

    # Test 6: The learner's lessons hold only the second lesson, now current
    sync_state(base_url, token, course["id"])
    page = list_lessons(base_url, token, course["id"], limit=20)
    actual = [(e["node"]["id"], e["node"]["is_current"]) for e in page["edges"]]
    assert actual == [(second_lesson["id"], True)], f"Expected only the second lesson, current, got {actual}"
    log_step("  ✓ Learner's lessons drop the deleted lesson - success")

    log_step("✓ DeleteLesson tests passed")


def get_review_response(base_url: str, token: str, params: Dict[str, Any]) -> requests.Response:
    response = requests.get(
        f"{base_url}/review",
        params=params,
        headers=auth_headers(token),
        timeout=TIMEOUT,
    )
    log_response("GET /review", response)
    return response


def review_answers_response(base_url: str, token: str, answer_ids: List[str]) -> requests.Response:
    response = requests.post(
        f"{base_url}/review/answers",
        json={"answer_ids": answer_ids},
        headers=auth_headers(token),
        timeout=TIMEOUT,
    )
    log_response("POST /review/answers", response)
    return response


def run_lesson_answers_review_test(base_url: str, token: str, deck_id: str) -> None:
    """Test that answering a lesson deck makes its cards available in GET /review."""
    log_step("")
    log_step("Testing review after answering lessons...")

    expected = sorted(card["id"] for card in get_deck(base_url, token, deck_id)["cards"])

    # The course service sends lesson answers to the dealer in the background, so they may
    # not be there yet on a slow machine. Give it a few tries before calling it a failure.
    for _ in range(10):
        response = get_review_response(base_url, token, {"deck_id": deck_id})
        assert response.status_code == 200, f"Expected 200, got {response.status_code}"
        actual = sorted(card["id"] for card in response.json()["cards"])
        if actual == expected:
            break
        time.sleep(0.2)
    assert actual == expected, f"Expected {expected}, got {actual}"
    log_step("  ✓ Review of a lesson deck returns its answered cards - success")


def run_review_tests(base_url: str) -> None:
    """Test GET /review and POST /review/answers with a user who has answered nothing yet."""
    log_step("")
    log_step("Testing review endpoints...")

    token = create_user(base_url)["token"]
    answered_deck = create_deck(base_url, token, "review-answered")
    course = create_course(base_url, token, "review-course", "review-course description")
    lesson = create_lesson(base_url, token, course["id"], 1, "review-lesson", answered_deck["id"])
    enroll_course(base_url, token, course["id"])

    response = get_review_response(base_url, token, {})
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    assert response.json()["cards"] == [], f"Expected no cards, got {response.json()}"
    log_step("  ✓ Review before answering is empty - success")

    # First answer is the correct one, see create_deck.
    answer_ids = [card["possible_answers"][0]["id"] for card in answered_deck["cards"]]

    # Cards only enter the review pool through a lesson, so these are rejected for now.
    response = review_answers_response(base_url, token, answer_ids)
    assert response.status_code == 404, f"Expected 404, got {response.status_code}"
    response = get_review_response(base_url, token, {})
    assert response.json()["cards"] == [], f"Expected no cards, got {response.json()}"
    log_step("  ✓ Review answers for cards never answered in a lesson return 404 - success")

    answer_deck(base_url, token, course["id"], lesson["id"], answered_deck)

    # 4 answered cards is fewer than the default max_cards, so every review returns all
    # of them whatever their level. The dealer shuffles them, hence the sort.
    expected = sorted(card["id"] for card in answered_deck["cards"])

    # The course service sends lesson answers to the dealer in the background, so they may
    # not be there yet on a slow machine. Give it a few tries before calling it a failure.
    for _ in range(10):
        response = get_review_response(base_url, token, {"deck_id": answered_deck["id"]})
        assert response.status_code == 200, f"Expected 200, got {response.status_code}"
        actual = sorted(card["id"] for card in response.json()["cards"])
        if actual == expected:
            break
        time.sleep(0.2)
    assert actual == expected, f"Expected {expected}, got {actual}"
    log_step("  ✓ Review of a deck answered in a lesson returns its cards - success")

    response = get_review_response(base_url, token, {})
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    actual = sorted(card["id"] for card in response.json()["cards"])
    assert actual == expected, f"Expected {expected}, got {actual}"
    log_step("  ✓ Review without scope returns all answered cards - success")

    response = get_review_response(base_url, token, {"course_id": course["id"]})
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    actual = sorted(card["id"] for card in response.json()["cards"])
    assert actual == expected, f"Expected {expected}, got {actual}"
    actual_deck_ids = {card["deck_id"] for card in response.json()["cards"]}
    assert actual_deck_ids == {answered_deck["id"]}, f"Expected deck_id {answered_deck['id']}, got {actual_deck_ids}"
    log_step("  ✓ Review of the course returns its lesson's cards, with their deck ID - success")

    response = get_review_response(base_url, token, {"lesson_id": lesson["id"]})
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    actual = sorted(card["id"] for card in response.json()["cards"])
    assert actual == expected, f"Expected {expected}, got {actual}"
    log_step("  ✓ Review of the lesson returns its deck's cards - success")

    response = get_review_response(base_url, token, {"course_id": "00000000-0000-0000-0000-000000000000"})
    assert response.status_code == 404, f"Expected 404, got {response.status_code}"
    log_step("  ✓ Review of an unknown course returns 404 - success")

    response = get_review_response(base_url, token, {"lesson_id": "00000000-0000-0000-0000-000000000000"})
    assert response.status_code == 404, f"Expected 404, got {response.status_code}"
    log_step("  ✓ Review of an unknown lesson returns 404 - success")

    response = get_review_response(base_url, token, {"deck_id": answered_deck["id"], "course_id": course["id"]})
    assert response.status_code == 400, f"Expected 400, got {response.status_code}"
    log_step("  ✓ Review with two scopes returns 400 - success")

    response = review_answers_response(base_url, token, answer_ids)
    assert response.status_code == 204, f"Expected 204, got {response.status_code}"
    log_step("  ✓ Review answers for cards in the pool - success")

    response = get_review_response(base_url, token, {"max_cards": 1})
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    assert len(response.json()["cards"]) == 1, f"Expected 1 card, got {response.json()}"
    log_step("  ✓ Review honours max_cards - success")

    response = get_review_response(base_url, token, {"max_cards": 0})
    assert response.status_code == 400, f"Expected 400, got {response.status_code}"
    log_step("  ✓ Review with max_cards=0 returns 400 - success")

    response = get_review_response(base_url, token, {"deck_id": "invalid-uuid"})
    assert response.status_code == 400, f"Expected 400, got {response.status_code}"
    log_step("  ✓ Review with invalid deck_id returns 400 - success")

    response = review_answers_response(base_url, token, ["00000000-0000-0000-0000-000000000000"])
    assert response.status_code == 404, f"Expected 404, got {response.status_code}"
    log_step("  ✓ Store unknown answer returns 404 - success")

    log_step("✓ Review tests passed")


def create_card_response(base_url: str, token: str, deck_id: str, card: Dict[str, Any]) -> requests.Response:
    response = requests.post(
        f"{base_url}/decks/{deck_id}/cards",
        json=card,
        headers=auth_headers(token),
        timeout=TIMEOUT,
    )
    log_response("POST /decks/{deckId}/cards", response)
    return response


def make_card_payload(title: str) -> Dict[str, Any]:
    return {
        "title": title,
        "kind": "single_choice",
        "explanation": f"{title} explanation",
        "possible_answers": [
            {"text": f"{title} right", "is_correct": True},
            {"text": f"{title} wrong", "is_correct": False},
        ],
    }


def run_create_card_tests(base_url: str, token: str) -> None:
    """Test POST /decks/{deckId}/cards, including that the cached deck shows the new card."""
    log_step("")
    log_step("Testing CreateCard endpoint...")

    deck = create_deck(base_url, token, "deck-for-new-card")

    # Reading the deck caches it, so the next read proves the cache was cleared.
    get_deck(base_url, token, deck["id"])

    response = create_card_response(base_url, token, deck["id"], make_card_payload("new-card"))
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    actual = response.json()
    assert actual["deck_id"] == deck["id"], f"Expected deck_id {deck['id']}, got {actual['deck_id']}"
    assert actual["title"] == "new-card", f"Expected title new-card, got {actual['title']}"
    assert actual["kind"] == "single_choice", f"Expected kind single_choice, got {actual['kind']}"
    assert actual["explanation"] == "new-card explanation", f"Unexpected explanation {actual['explanation']}"
    answers = sorted((a["text"], a["is_correct"], a["card_id"]) for a in actual["possible_answers"])
    expected_answers = [("new-card right", True, actual["id"]), ("new-card wrong", False, actual["id"])]
    assert answers == expected_answers, f"Expected {expected_answers}, got {answers}"
    log_step("  ✓ Create card returns the new card - success")

    deck_after = get_deck(base_url, token, deck["id"])
    card_ids = [card["id"] for card in deck_after["cards"]]
    assert actual["id"] in card_ids, f"New card {actual['id']} missing from deck, got {card_ids}"
    assert len(card_ids) == len(deck["cards"]) + 1, f"Expected {len(deck['cards']) + 1} cards, got {len(card_ids)}"
    log_step("  ✓ Deck read after adding a card includes it - success")

    response = create_card_response(base_url, token, "00000000-0000-0000-0000-000000000000", make_card_payload("orphan"))
    assert response.status_code == 404, f"Expected 404, got {response.status_code}"
    log_step("  ✓ Create card on an unknown deck returns 404 - success")

    invalid_card = {"title": "no-answers", "kind": "single_choice", "possible_answers": []}
    response = create_card_response(base_url, token, deck["id"], invalid_card)
    assert response.status_code == 400, f"Expected 400, got {response.status_code}"
    log_step("  ✓ Create card without answers returns 400 - success")

    log_step("✓ CreateCard tests passed")


def run_new_card_reopens_lesson_test(base_url: str) -> None:
    """Test that adding a card to a completed lesson's deck reopens the lesson after a sync."""
    log_step("")
    log_step("Testing that a new card reopens a completed lesson...")

    token = create_user(base_url)["token"]
    deck = create_deck(base_url, token, "deck-reopened")
    course = create_course(base_url, token, "reopen-course", "reopen-course description")
    lesson = create_lesson(base_url, token, course["id"], 1, "reopen-lesson", deck["id"])
    enroll_course(base_url, token, course["id"])
    answer_deck(base_url, token, course["id"], lesson["id"], deck)

    lesson_progress = get_lesson_state(base_url, token, course["id"], lesson["id"])["lesson_state"][lesson["id"]]
    assert lesson_progress["is_completed"] is True, "Lesson should be completed before the new card"
    log_step("  ✓ Lesson is completed - success")

    response = create_card_response(base_url, token, deck["id"], make_card_payload("late-card"))
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    new_card_id = response.json()["id"]

    sync_state(base_url, token, course["id"])

    lesson_progress = get_lesson_state(base_url, token, course["id"], lesson["id"])["lesson_state"][lesson["id"]]
    assert lesson_progress["is_completed"] is False, "Lesson should be reopened by the new card"
    deck_progress = lesson_progress["decks"][deck["id"]]
    assert deck_progress["is_completed"] is False, "Deck should be reopened by the new card"
    assert deck_progress["cards"][new_card_id]["is_completed"] is False, "New card should be unanswered"
    log_step("  ✓ New card reopens the lesson, unanswered - success")

    log_step("✓ New card reopens lesson test passed")


def run_new_deck_reopens_lesson_test(base_url: str) -> None:
    """Test that adding a deck to a completed lesson reopens the lesson after a sync."""
    log_step("")
    log_step("Testing that a new deck reopens a completed lesson...")

    token = create_user(base_url)["token"]
    first_deck = create_deck(base_url, token, "deck-first")
    second_deck = create_deck(base_url, token, "deck-added-later")
    course = create_course(base_url, token, "reopen-deck-course", "reopen-deck-course description")
    lesson = create_lesson(base_url, token, course["id"], 1, "reopen-deck-lesson", first_deck["id"])
    enroll_course(base_url, token, course["id"])
    answer_deck(base_url, token, course["id"], lesson["id"], first_deck)

    lesson_progress = get_lesson_state(base_url, token, course["id"], lesson["id"])["lesson_state"][lesson["id"]]
    assert lesson_progress["is_completed"] is True, "Lesson should be completed before the new deck"
    log_step("  ✓ Lesson is completed - success")

    # A lesson's decks are the ![deck](id) references in its body, so this adds second_deck to the lesson.
    new_body = f"Lesson with decks ![deck]({first_deck['id']}) and ![deck]({second_deck['id']})"
    response = update_lesson(base_url, token, course["id"], lesson["id"], {"body": new_body})
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"

    sync_state(base_url, token, course["id"])

    lesson_progress = get_lesson_state(base_url, token, course["id"], lesson["id"])["lesson_state"][lesson["id"]]
    assert lesson_progress["is_completed"] is False, "Lesson should be reopened by the new deck"
    assert lesson_progress["decks"][first_deck["id"]]["is_completed"] is True, "First deck should stay completed"
    new_deck_progress = lesson_progress["decks"][second_deck["id"]]
    assert new_deck_progress["is_completed"] is False, "New deck should not be completed"
    expected_card_ids = sorted(card["id"] for card in second_deck["cards"])
    actual_card_ids = sorted(new_deck_progress["cards"].keys())
    assert actual_card_ids == expected_card_ids, f"Expected cards {expected_card_ids}, got {actual_card_ids}"
    actual_completed = {card["is_completed"] for card in new_deck_progress["cards"].values()}
    assert actual_completed == {False}, f"New deck's cards should all be unanswered, got {actual_completed}"
    log_step("  ✓ New deck reopens the lesson, with its cards unanswered - success")

    log_step("✓ New deck reopens lesson test passed")


def run_update_course_tests(base_url: str, token: str, course_id: str) -> None:
    """Test PATCH /courses/{courseId} endpoint."""
    log_step("")
    log_step("Testing UpdateCourse endpoint...")

    # Test 1: Update only title
    response = update_course(base_url, token, course_id, {
                             "title": "Updated Course Title"})
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    updated = response.json()
    assert updated["title"] == "Updated Course Title", "Title should be updated"
    log_step("  ✓ Update title only - success")

    # Test 2: Update only description
    response = update_course(base_url, token, course_id, {
                             "description": "Updated description"})
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    updated = response.json()
    assert updated["description"] == "Updated description", "Description should be updated"
    assert updated["title"] == "Updated Course Title", "Title should remain from previous update"
    log_step("  ✓ Update description only - success")

    # Test 3: Update order
    response = update_course(base_url, token, course_id, {"order": 100})
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    updated = response.json()
    assert updated["order"] == 100, "Order should be updated"
    log_step("  ✓ Update order only - success")

    # Test 4: Update multiple fields
    response = update_course(base_url, token, course_id, {
        "title": "Final Title",
        "description": "Final Description",
        "order": 1
    })
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    updated = response.json()
    assert updated["title"] == "Final Title"
    assert updated["description"] == "Final Description"
    assert updated["order"] == 1
    log_step("  ✓ Update multiple fields - success")

    # Test 5: Re-fetch course to verify persistence
    log_step("  Verifying persistence by re-fetching course...")
    get_response = requests.get(
        f"{base_url}/courses/{course_id}",
        headers=auth_headers(token),
        timeout=TIMEOUT,
    )
    assert get_response.status_code == 200
    fetched = get_response.json()
    assert fetched[
        "title"] == "Final Title", f"Title not persisted: got {fetched.get('title')}"
    assert fetched[
        "description"] == "Final Description", f"Description not persisted: got {fetched.get('description')}"
    assert fetched["order"] == 1, f"Order not persisted: got {fetched.get('order')}"
    log_step("  ✓ Re-fetch confirms persistence - success")

    # Test 6: Empty request should return 400
    response = update_course(base_url, token, course_id, {})
    assert response.status_code == 400, f"Expected 400 for empty request, got {response.status_code}"
    log_step("  ✓ Empty request returns 400 - success")

    # Test 7: Non-existent course should return 404
    fake_uuid = "00000000-0000-0000-0000-000000000000"
    response = update_course(base_url, token, fake_uuid, {
                             "title": "Won't work"})
    assert response.status_code == 404, f"Expected 404 for non-existent course, got {response.status_code}"
    log_step("  ✓ Non-existent course returns 404 - success")

    # Test 8: Empty string is allowed (not treated as missing)
    response = update_course(base_url, token, course_id, {"title": ""})
    assert response.status_code == 200, f"Expected 200 for empty string, got {response.status_code}"
    updated = response.json()
    assert updated["title"] == "", "Title should be empty string"
    log_step("  ✓ Empty string allowed - success")

    # Restore title for other tests
    update_course(base_url, token, course_id, {"title": "course-0"})

    log_step("✓ UpdateCourse tests passed")


def run_update_lesson_tests(base_url: str, token: str, course_id: str, lesson: Dict[str, Any], deck_id: str) -> None:
    """Test PATCH /courses/{courseId}/lessons/{lessonId} endpoint."""
    log_step("")
    log_step("Testing UpdateLesson endpoint...")

    lesson_id = lesson["id"]

    # Test 1: Update only title
    response = update_lesson(base_url, token, course_id, lesson_id, {
                             "title": "Updated Lesson Title"})
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    updated = response.json()
    assert updated["title"] == "Updated Lesson Title", "Title should be updated"
    log_step("  ✓ Update title only - success")

    # Test 2: Update only description
    response = update_lesson(base_url, token, course_id, lesson_id, {
                             "description": "Updated lesson desc"})
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    updated = response.json()
    assert updated["description"] == "Updated lesson desc", "Description should be updated"
    assert updated["title"] == "Updated Lesson Title", "Title should remain from previous update"
    log_step("  ✓ Update description only - success")

    # Test 3: Update order
    response = update_lesson(base_url, token, course_id,
                             lesson_id, {"order": 99})
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    updated = response.json()
    assert updated["order"] == 99, "Order should be updated"
    log_step("  ✓ Update order only - success")

    # Test 4: Update body with valid deck reference (re-validates deck)
    new_body = f"New lesson body with deck ![deck]({deck_id})"
    response = update_lesson(base_url, token, course_id,
                             lesson_id, {"body": new_body})
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    updated = response.json()
    assert updated["body"] == new_body, "Body should be updated"
    log_step("  ✓ Update body with valid deck reference - success")

    # Test 5: Update body with invalid deck reference should fail
    invalid_body = "Body with invalid deck ![deck](00000000-0000-0000-0000-000000000000)"
    response = update_lesson(base_url, token, course_id,
                             lesson_id, {"body": invalid_body})
    # This should fail because the deck doesn't exist
    assert response.status_code in (
        400, 404, 500), f"Expected error for invalid deck reference, got {response.status_code}"
    log_step("  ✓ Update body with invalid deck reference rejected - success")

    # Test 6: Update multiple fields
    response = update_lesson(base_url, token, course_id, lesson_id, {
        "title": "Final Lesson Title",
        "description": "Final lesson desc",
        "order": 1
    })
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    updated = response.json()
    assert updated["title"] == "Final Lesson Title"
    assert updated["description"] == "Final lesson desc"
    assert updated["order"] == 1
    log_step("  ✓ Update multiple fields - success")

    # Test 7: Re-fetch lesson to verify persistence
    log_step("  Verifying persistence by re-fetching the lesson...")
    response = get_lesson_response(base_url, token, course_id, lesson_id)
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    fetched_lesson = response.json()
    assert fetched_lesson["title"] == "Final Lesson Title", "Title not persisted"
    assert fetched_lesson["description"] == "Final lesson desc", "Description not persisted"
    assert fetched_lesson["order"] == 1, "Order not persisted"
    assert fetched_lesson["body"] == new_body, "Body not persisted"
    log_step("  ✓ Re-fetch confirms persistence - success")

    # Test 8: Empty request should return 400
    response = update_lesson(base_url, token, course_id, lesson_id, {})
    assert response.status_code == 400, f"Expected 400 for empty request, got {response.status_code}"
    log_step("  ✓ Empty request returns 400 - success")

    # Test 9: Non-existent lesson should return 404
    fake_uuid = "00000000-0000-0000-0000-000000000000"
    response = update_lesson(base_url, token, course_id,
                             fake_uuid, {"title": "Won't work"})
    assert response.status_code == 404, f"Expected 404 for non-existent lesson, got {response.status_code}"
    log_step("  ✓ Non-existent lesson returns 404 - success")

    log_step("✓ UpdateLesson tests passed")


def run_update_deck_tests(base_url: str, token: str, deck_id: str) -> None:
    """Test PATCH /decks/{deckId} endpoint."""
    log_step("")
    log_step("Testing UpdateDeck endpoint...")

    # Test 1: Update only title
    response = update_deck(base_url, token, deck_id, {"title": "Updated Deck Title"})
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    updated = response.json()
    assert updated["title"] == "Updated Deck Title", "Title should be updated"
    log_step("  ✓ Update title only - success")

    # Test 2: Update only description
    response = update_deck(base_url, token, deck_id, {"description": "Updated deck desc"})
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    updated = response.json()
    assert updated["description"] == "Updated deck desc", "Description should be updated"
    assert updated["title"] == "Updated Deck Title", "Title should remain from previous update"
    log_step("  ✓ Update description only - success")

    # Test 3: Update multiple fields
    response = update_deck(base_url, token, deck_id, {
        "title": "Final Deck Title",
        "description": "Final deck desc"
    })
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    updated = response.json()
    assert updated["title"] == "Final Deck Title"
    assert updated["description"] == "Final deck desc"
    log_step("  ✓ Update multiple fields - success")

    # Test 4: Re-fetch deck to verify persistence
    log_step("  Verifying persistence by re-fetching deck...")
    get_response = get_deck_response(base_url, token, deck_id)
    assert get_response.status_code == 200
    fetched = get_response.json()
    assert fetched["title"] == "Final Deck Title", "Title not persisted"
    assert fetched["description"] == "Final deck desc", "Description not persisted"
    log_step("  ✓ Re-fetch confirms persistence - success")

    # Test 5: Empty request should return 400
    response = update_deck(base_url, token, deck_id, {})
    assert response.status_code == 400, f"Expected 400 for empty request, got {response.status_code}"
    log_step("  ✓ Empty request returns 400 - success")

    # Test 6: Non-existent deck should return 404
    fake_uuid = "00000000-0000-0000-0000-000000000000"
    response = update_deck(base_url, token, fake_uuid, {"title": "Won't work"})
    assert response.status_code == 404, f"Expected 404 for non-existent deck, got {response.status_code}"
    log_step("  ✓ Non-existent deck returns 404 - success")

    log_step("✓ UpdateDeck tests passed")


def run_update_card_tests(base_url: str, token: str, deck_id: str, card_id: str) -> None:
    """Test PATCH /decks/{deckId}/cards/{cardId} endpoint."""
    log_step("")
    log_step("Testing UpdateCard endpoint...")

    # Test 1: Update only title
    response = update_card(base_url, token, deck_id, card_id, {"title": "Updated Card Title"})
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    updated = response.json()
    assert updated["title"] == "Updated Card Title", "Title should be updated"
    log_step("  ✓ Update title only - success")

    # Test 2: Update explanation
    response = update_card(base_url, token, deck_id, card_id, {"explanation": "Updated explanation"})
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    updated = response.json()
    assert updated["explanation"] == "Updated explanation", "Explanation should be updated"
    log_step("  ✓ Update explanation only - success")

    # Test 3: Update kind (valid value)
    response = update_card(base_url, token, deck_id, card_id, {"kind": "fill_in_the_blanks"})
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    updated = response.json()
    assert updated["kind"] == "fill_in_the_blanks", "Kind should be updated"
    log_step("  ✓ Update kind only - success")

    # Test 4: Update kind with invalid value should fail
    response = update_card(base_url, token, deck_id, card_id, {"kind": "invalid_kind"})
    assert response.status_code == 400, f"Expected 400 for invalid kind, got {response.status_code}"
    log_step("  ✓ Invalid kind returns 400 - success")

    # Test 5: Update multiple fields
    response = update_card(base_url, token, deck_id, card_id, {
        "title": "Final Card Title",
        "explanation": "Final explanation",
        "kind": "single_choice"
    })
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    updated = response.json()
    assert updated["title"] == "Final Card Title"
    assert updated["explanation"] == "Final explanation"
    assert updated["kind"] == "single_choice"
    log_step("  ✓ Update multiple fields - success")

    # Test 6: Re-fetch deck to verify card persistence
    log_step("  Verifying persistence by re-fetching deck...")
    get_response = get_deck_response(base_url, token, deck_id)
    assert get_response.status_code == 200
    fetched_deck = get_response.json()
    fetched_card = next((c for c in fetched_deck["cards"] if c["id"] == card_id), None)
    assert fetched_card is not None, "Card should be found in deck"
    assert fetched_card["title"] == "Final Card Title", "Title not persisted"
    assert fetched_card["explanation"] == "Final explanation", "Explanation not persisted"
    log_step("  ✓ Re-fetch confirms persistence - success")

    # Test 7: Empty request should return 400
    response = update_card(base_url, token, deck_id, card_id, {})
    assert response.status_code == 400, f"Expected 400 for empty request, got {response.status_code}"
    log_step("  ✓ Empty request returns 400 - success")

    # Test 8: Non-existent card should return 404
    fake_uuid = "00000000-0000-0000-0000-000000000000"
    response = update_card(base_url, token, deck_id, fake_uuid, {"title": "Won't work"})
    assert response.status_code == 404, f"Expected 404 for non-existent card, got {response.status_code}"
    log_step("  ✓ Non-existent card returns 404 - success")

    log_step("✓ UpdateCard tests passed")


def run_update_answer_tests(base_url: str, token: str, deck_id: str, card_id: str, answer_id: str) -> None:
    """Test PATCH /decks/{deckId}/cards/{cardId}/answers/{answerId} endpoint."""
    log_step("")
    log_step("Testing UpdateAnswer endpoint...")

    # Test 1: Update only text
    response = update_answer(base_url, token, deck_id, card_id, answer_id, {"text": "Updated Answer Text"})
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    updated = response.json()
    assert updated["text"] == "Updated Answer Text", "Text should be updated"
    log_step("  ✓ Update text only - success")

    # Test 2: Update is_correct
    response = update_answer(base_url, token, deck_id, card_id, answer_id, {"is_correct": True})
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    updated = response.json()
    assert updated["is_correct"] == True, "is_correct should be updated"
    log_step("  ✓ Update is_correct only - success")

    # Test 3: Update multiple fields
    response = update_answer(base_url, token, deck_id, card_id, answer_id, {
        "text": "Final Answer Text",
        "is_correct": False
    })
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    updated = response.json()
    assert updated["text"] == "Final Answer Text"
    assert updated["is_correct"] == False
    log_step("  ✓ Update multiple fields - success")

    # Test 4: Re-fetch deck to verify answer persistence
    log_step("  Verifying persistence by re-fetching deck...")
    get_response = get_deck_response(base_url, token, deck_id)
    assert get_response.status_code == 200
    fetched_deck = get_response.json()
    fetched_card = next((c for c in fetched_deck["cards"] if c["id"] == card_id), None)
    assert fetched_card is not None, "Card should be found in deck"
    fetched_answer = next((a for a in fetched_card["possible_answers"] if a["id"] == answer_id), None)
    assert fetched_answer is not None, "Answer should be found in card"
    assert fetched_answer["text"] == "Final Answer Text", "Text not persisted"
    assert fetched_answer["is_correct"] == False, "is_correct not persisted"
    log_step("  ✓ Re-fetch confirms persistence - success")

    # Test 5: Empty request should return 400
    response = update_answer(base_url, token, deck_id, card_id, answer_id, {})
    assert response.status_code == 400, f"Expected 400 for empty request, got {response.status_code}"
    log_step("  ✓ Empty request returns 400 - success")

    log_step("✓ UpdateAnswer tests passed")


def run_get_lesson_tests(base_url: str, token: str, course_id: str, lesson: Dict[str, Any], other_course_id: str) -> None:
    """Test GET /courses/{courseId}/lessons/{lessonId} endpoint."""
    log_step("")
    log_step("Testing GetLesson endpoint...")

    # Test 1: Fetch existing lesson
    response = get_lesson_response(base_url, token, course_id, lesson["id"])
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    assert response.json() == lesson, "Fetched lesson should match the created one"
    log_step("  ✓ Get existing lesson - success")

    # Test 2: Fetch non-existent lesson
    fake_uuid = "00000000-0000-0000-0000-000000000000"
    response = get_lesson_response(base_url, token, course_id, fake_uuid)
    assert response.status_code == 404, f"Expected 404 for non-existent lesson, got {response.status_code}"
    log_step("  ✓ Get non-existent lesson returns 404 - success")

    # Test 3: Fetch a lesson through another course
    response = get_lesson_response(base_url, token, other_course_id, lesson["id"])
    assert response.status_code == 404, f"Expected 404 for a lesson from another course, got {response.status_code}"
    log_step("  ✓ Get lesson through another course returns 404 - success")

    log_step("✓ GetLesson tests passed")


def run_lessons_require_enrolment_test(base_url: str, course_id: str) -> None:
    """Test that listing a course's lessons needs the user to be enrolled."""
    log_step("")
    log_step("Testing lessons list for a user who isn't enrolled...")

    token = create_user(base_url)["token"]
    response = requests.get(
        f"{base_url}/courses/{course_id}/lessons",
        headers=auth_headers(token),
        timeout=TIMEOUT,
    )
    assert response.status_code == 404, f"Expected 404 for a user who isn't enrolled, got {response.status_code}"
    log_step("✓ Lessons list returns 404 for a user who isn't enrolled")


def admin_get_lessons_response(base_url: str, headers: Dict[str, str], course_id: str, params: Dict[str, Any]) -> requests.Response:
    response = requests.get(
        f"{base_url}/admin/courses/{course_id}/lessons",
        headers=headers,
        params=params,
        timeout=TIMEOUT,
    )
    log_response("GET /admin/courses/{courseId}/lessons", response)
    return response


def run_admin_get_lessons_tests(base_url: str, course_id: str, lessons: List[Dict[str, Any]]) -> None:
    """Test GET /admin/courses/{courseId}/lessons with a user who isn't enrolled."""
    log_step("")
    log_step("Testing AdminGetLessons endpoint...")

    token = create_user(base_url)["token"]

    # Test 1: First page, with bodies and without progress
    response = admin_get_lessons_response(base_url, auth_headers(token), course_id, {"first": 2})
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    page1 = response.json()
    assert [e["node"] for e in page1["edges"]] == lessons[0:2], "First page should hold the first two lessons, with bodies"
    assert page1["page_info"]["has_next_page"] is True, "First page should have a next page"
    log_step("  ✓ First page - success")

    # Test 2: Second page
    response = admin_get_lessons_response(base_url, auth_headers(token), course_id, {"first": 2, "after": page1["page_info"]["end_cursor"]})
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    page2 = response.json()
    assert [e["node"] for e in page2["edges"]] == lessons[2:4], "Second page should hold lessons 3 and 4"
    assert page2["page_info"]["has_next_page"] is True, "Second page should have a next page"
    log_step("  ✓ Second page - success")

    # Test 3: Last page
    response = admin_get_lessons_response(base_url, auth_headers(token), course_id, {"first": 2, "after": page2["page_info"]["end_cursor"]})
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    page3 = response.json()
    assert [e["node"] for e in page3["edges"]] == lessons[4:5], "Last page should hold lesson 5"
    assert page3["page_info"]["has_next_page"] is False, "Last page should have no next page"
    log_step("  ✓ Last page - success")

    # Test 4: Back from the last page
    response = admin_get_lessons_response(base_url, auth_headers(token), course_id, {"last": 2, "before": page3["page_info"]["start_cursor"]})
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    back = response.json()
    assert [e["node"] for e in back["edges"]] == lessons[2:4], "Paging back from lesson 5 should give lessons 3 and 4"
    log_step("  ✓ Backward page - success")

    # Test 5: Without the admin header
    response = admin_get_lessons_response(base_url, {"Authorization": f"Bearer {token}"}, course_id, {})
    assert response.status_code == 403, f"Expected 403 without the admin header, got {response.status_code}"
    log_step("  ✓ Without admin header returns 403 - success")

    # Test 6: Course that doesn't exist
    response = admin_get_lessons_response(base_url, auth_headers(token), "00000000-0000-0000-0000-000000000000", {})
    assert response.status_code == 404, f"Expected 404 for a course that doesn't exist, got {response.status_code}"
    log_step("  ✓ Missing course returns 404 - success")

    # Test 7: Malformed course ID
    response = admin_get_lessons_response(base_url, auth_headers(token), "not-a-uuid", {})
    assert response.status_code == 400, f"Expected 400 for a malformed course ID, got {response.status_code}"
    log_step("  ✓ Malformed course ID returns 400 - success")

    log_step("✓ AdminGetLessons tests passed")


def run_backoffice_tests(base_url: str, token: str, course_id: str, created_course_ids: List[str]) -> None:
    """Test backoffice endpoints (admin-only)."""
    log_step("")
    log_step("Testing Backoffice endpoints...")

    # Test 1: Get students progress
    log_step("  Testing GET /admin/courses/{courseId}/students/progress...")
    response = requests.get(
        f"{base_url}/admin/courses/{course_id}/students/progress",
        headers=auth_headers(token),
        timeout=TIMEOUT,
    )
    log_response("GET /admin/courses/{courseId}/students/progress", response)
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    data = response.json()
    assert "edges" in data, "Response should have 'edges' field"
    assert "page_info" in data, "Response should have 'page_info' field"
    assert "total_enrolled" in data, "Response should have 'total_enrolled' field"
    assert data["total_enrolled"] >= 1, "Should have at least 1 enrolled student"
    log_step("  ✓ Get students progress - success")

    # Extract the user_id from the first enrolled student for subsequent tests
    assert len(data["edges"]) > 0, "Should have at least one student edge"
    student_node = data["edges"][0]["node"]
    user_id = student_node["user_id"]
    assert user_id, "Student should have user_id"
    # Verify user enrichment - username and nick should be present
    assert "username" in student_node, "Student should have username from user enrichment"
    assert student_node["username"], "Username should not be empty"
    log_step(f"  Found enrolled user: {user_id} (username: {student_node['username']})")

    # Test 2: Get students progress with detailed=true
    log_step("  Testing GET /admin/courses/{courseId}/students/progress?detailed=true...")
    response = requests.get(
        f"{base_url}/admin/courses/{course_id}/students/progress?detailed=true",
        headers=auth_headers(token),
        timeout=TIMEOUT,
    )
    log_response("GET /admin/courses/{courseId}/students/progress?detailed=true", response)
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    data = response.json()
    if len(data["edges"]) > 0:
        student = data["edges"][0]["node"]
        assert "lesson_details" in student, "Student should have lesson_details when detailed=true"
    log_step("  ✓ Get students progress with details - success")

    # Test 3: Get user state
    log_step("  Testing GET /admin/courses/{courseId}/users/{userId}/state...")
    response = requests.get(
        f"{base_url}/admin/courses/{course_id}/users/{user_id}/state",
        headers=auth_headers(token),
        timeout=TIMEOUT,
    )
    log_response("GET /admin/courses/{courseId}/users/{userId}/state", response)
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    data = response.json()
    assert "user_id" in data, "Response should have 'user_id' field"
    assert "course_id" in data, "Response should have 'course_id' field"
    assert "state_json" in data, "Response should have 'state_json' field"
    assert "current_lesson_id" in data, "Response should have 'current_lesson_id' field"
    # Store for later
    original_state = data["state_json"]
    original_lesson_id = data["current_lesson_id"]
    log_step("  ✓ Get user state - success")

    # Test 4: Set user state (and restore it back)
    log_step("  Testing PUT /admin/courses/{courseId}/users/{userId}/state...")
    response = requests.put(
        f"{base_url}/admin/courses/{course_id}/users/{user_id}/state",
        headers=auth_headers(token),
        json={
            "state_json": original_state,
            "current_lesson_id": original_lesson_id,
        },
        timeout=TIMEOUT,
    )
    log_response("PUT /admin/courses/{courseId}/users/{userId}/state", response)
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    data = response.json()
    # Response returns the updated state
    assert "user_id" in data, "Response should have 'user_id' field"
    assert "state_json" in data, "Response should have 'state_json' field"
    log_step("  ✓ Set user state - success")

    # Test 5: Set user state with sync_after=true
    log_step("  Testing PUT /admin/courses/{courseId}/users/{userId}/state?sync=true...")
    response = requests.put(
        f"{base_url}/admin/courses/{course_id}/users/{user_id}/state?sync=true",
        headers=auth_headers(token),
        json={
            "state_json": original_state,
            "current_lesson_id": original_lesson_id,
        },
        timeout=TIMEOUT,
    )
    log_response("PUT /admin/courses/{courseId}/users/{userId}/state?sync=true", response)
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    log_step("  ✓ Set user state with sync - success")

    # Test 6: Get user state for non-enrolled user (should 404)
    fake_user_id = "00000000-0000-0000-0000-000000000000"
    log_step("  Testing GET user state for non-enrolled user...")
    response = requests.get(
        f"{base_url}/admin/courses/{course_id}/users/{fake_user_id}/state",
        headers=auth_headers(token),
        timeout=TIMEOUT,
    )
    log_response("GET /admin/courses/{courseId}/users/{fakeUserId}/state", response)
    assert response.status_code == 404, f"Expected 404 for non-enrolled user, got {response.status_code}"
    log_step("  ✓ Get state for non-enrolled user returns 404 - success")

    # Test 7: List all courses
    log_step("  Testing GET /admin/courses...")
    response = requests.get(
        f"{base_url}/admin/courses",
        params={"first": 1},
        headers=auth_headers(token),
        timeout=TIMEOUT,
    )
    log_response("GET /admin/courses?first=1", response)
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    data = response.json()
    assert "edges" in data, "Response should have 'edges' field"
    assert "page_info" in data, "Response should have 'page_info' field"
    total_count = data["total_count"]
    assert total_count >= len(created_course_ids), f"Expected at least {len(created_course_ids)} courses, got {total_count}"
    node = data["edges"][0]["node"]
    for field in ("id", "title", "description", "order"):
        assert field in node, f"Course node should have '{field}' field"
    log_step(f"  ✓ Get all courses - success (total_count={total_count})")

    # Previous runs leave courses with the same orders behind, so these pages cross duplicate orders.
    log_step("  Testing forward pagination over all courses...")
    response = requests.get(
        f"{base_url}/admin/courses",
        params={"first": total_count - 1},
        headers=auth_headers(token),
        timeout=TIMEOUT,
    )
    log_response(f"GET /admin/courses?first={total_count - 1}", response)
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    page1 = response.json()
    assert len(page1["edges"]) == total_count - 1, f"Expected {total_count - 1} edges, got {len(page1['edges'])}"
    assert page1["page_info"]["has_next_page"], "First page should have a next page"
    assert not page1["page_info"]["has_previous_page"], "First page should not have a previous page"

    response = requests.get(
        f"{base_url}/admin/courses",
        params={"first": total_count - 1, "after": page1["page_info"]["end_cursor"]},
        headers=auth_headers(token),
        timeout=TIMEOUT,
    )
    log_response("GET /admin/courses?after=<end_cursor>", response)
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    page2 = response.json()
    assert len(page2["edges"]) == 1, f"Expected 1 edge on last page, got {len(page2['edges'])}"
    assert not page2["page_info"]["has_next_page"], "Last page should not have a next page"

    forward_ids = [e["node"]["id"] for e in page1["edges"] + page2["edges"]]
    assert len(set(forward_ids)) == total_count, "Forward pagination should return every course exactly once"
    missing = set(created_course_ids) - set(forward_ids)
    assert not missing, f"Created courses missing from listing: {missing}"
    log_step("  ✓ Forward pagination returns every course once - success")

    log_step("  Testing backward pagination over all courses...")
    response = requests.get(
        f"{base_url}/admin/courses",
        params={"last": total_count - 1},
        headers=auth_headers(token),
        timeout=TIMEOUT,
    )
    log_response(f"GET /admin/courses?last={total_count - 1}", response)
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    page1 = response.json()
    assert len(page1["edges"]) == total_count - 1, f"Expected {total_count - 1} edges, got {len(page1['edges'])}"
    assert page1["page_info"]["has_previous_page"], "Last page should have a previous page"

    response = requests.get(
        f"{base_url}/admin/courses",
        params={"last": total_count - 1, "before": page1["page_info"]["start_cursor"]},
        headers=auth_headers(token),
        timeout=TIMEOUT,
    )
    log_response("GET /admin/courses?before=<start_cursor>", response)
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    page2 = response.json()
    assert len(page2["edges"]) == 1, f"Expected 1 edge on first page, got {len(page2['edges'])}"
    assert not page2["page_info"]["has_previous_page"], "First page should not have a previous page"

    backward_ids = [e["node"]["id"] for e in page2["edges"] + page1["edges"]]
    assert backward_ids == forward_ids, "Backward pagination should return the same order as forward"
    log_step("  ✓ Backward pagination matches forward order - success")

    log_step("  Testing GET /admin/courses without admin token...")
    response = requests.get(
        f"{base_url}/admin/courses",
        headers={"Authorization": f"Bearer {token}"},
        timeout=TIMEOUT,
    )
    log_response("GET /admin/courses (no admin token)", response)
    assert response.status_code == 403, f"Expected 403 without admin token, got {response.status_code}"
    log_step("  ✓ Missing admin token returns 403 - success")

    log_step("  Testing GET /admin/courses with invalid pagination...")
    response = requests.get(
        f"{base_url}/admin/courses",
        params={"first": "abc"},
        headers=auth_headers(token),
        timeout=TIMEOUT,
    )
    log_response("GET /admin/courses?first=abc", response)
    assert response.status_code == 400, f"Expected 400 for invalid pagination, got {response.status_code}"
    log_step("  ✓ Invalid pagination returns 400 - success")

    log_step("  Testing GET /admin/courses with invalid cursor...")
    response = requests.get(
        f"{base_url}/admin/courses",
        params={"after": "not-a-cursor"},
        headers=auth_headers(token),
        timeout=TIMEOUT,
    )
    log_response("GET /admin/courses?after=not-a-cursor", response)
    assert response.status_code == 400, f"Expected 400 for invalid cursor, got {response.status_code}"
    log_step("  ✓ Invalid cursor returns 400 - success")

    log_step("✓ Backoffice tests passed")


def test_basic_flow():
    init_log_file()

    base_url = load_base_url()
    log_step(f"Starting test against {base_url}")

    user = create_user(base_url)
    log_step(f"✓ User created: {user['username']}")
    token = user["token"]

    log_step("")
    course_id, lessons, lesson_deck_ids, all_course_ids = setup_test_data(
        base_url, token)

    log_step("")
    log_step(f"Enrolling in {len(all_course_ids)} courses...")
    for cid in all_course_ids:
        enroll_course(base_url, token, cid)
    log_step(f"✓ Enrolled in {len(all_course_ids)} courses")

    # Test get lesson individually
    run_get_lesson_tests(base_url, token, course_id, lessons[0], all_course_ids[1])

    run_lessons_require_enrolment_test(base_url, course_id)

    run_admin_get_lessons_tests(base_url, course_id, lessons)

    run_pagination_before_answering(base_url, token, course_id, lessons)

    run_focused_lessons_before_answering(base_url, token, course_id)

    # Run bodyless focused lessons test before answering
    run_bodyless_focused_lessons_test(base_url, token, course_id)

    first_lesson_id = lessons[0]["id"]

    run_lesson_state_before_answering(
        base_url, token, course_id, first_lesson_id)

    log_step("")
    log_step("Answering all decks...")
    complete_course(base_url, token, course_id, lessons, lesson_deck_ids)
    log_step(f"✓ Course completed: {len(lessons)} lesson(s)")

    log_step("")
    run_focused_lessons_after_answering(base_url, token, course_id)

    run_lesson_state_after_answering(
        base_url, token, course_id, first_lesson_id)

    run_backward_pagination_after_answering(base_url, token, course_id)

    run_enrolled_courses_test(base_url, token, course_id)

    sync_state(base_url, token, course_id)

    # Test backoffice endpoints (admin-only)
    run_backoffice_tests(base_url, token, course_id, all_course_ids)

    # Test update endpoints (admin-protected)
    run_update_course_tests(base_url, token, course_id)
    run_update_lesson_tests(base_url, token, course_id,
                            lessons[0], lesson_deck_ids[0])

    # Test deck/card/answer update endpoints
    deck_id = lesson_deck_ids[0]
    run_update_deck_tests(base_url, token, deck_id)

    # Fetch deck to get card and answer IDs for testing
    deck_response = get_deck_response(base_url, token, deck_id)
    assert deck_response.status_code == 200, f"Failed to fetch deck: {deck_response.status_code}"
    deck_data = deck_response.json()
    first_card = deck_data["cards"][0]
    card_id = first_card["id"]
    first_answer = first_card["possible_answers"][0]
    answer_id = first_answer["id"]

    run_update_card_tests(base_url, token, deck_id, card_id)
    run_update_answer_tests(base_url, token, deck_id, card_id, answer_id)

    # Test delete deck endpoint (creates its own deck)
    run_delete_deck_tests(base_url, token)

    run_delete_card_tests(base_url)

    run_delete_lesson_tests(base_url)

    run_delete_course_tests(base_url)

    run_create_card_tests(base_url, token)

    run_new_card_reopens_lesson_test(base_url)

    run_new_deck_reopens_lesson_test(base_url)

    run_lesson_answers_review_test(base_url, token, lesson_deck_ids[0])

    run_review_tests(base_url)

    log_step("")
    log_step("✅ Test passed!")
    log_step(f"📝 Detailed logs written to: {LOG_FILE}")

    # Write HAR file
    write_har_file()
    log_step(f"📊 HAR file written to: {HAR_FILE}")
