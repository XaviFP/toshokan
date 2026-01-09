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


def auth_headers(token: str) -> Dict[str, str]:
    return {"Authorization": f"Bearer {token}"}


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
    """Log error response to console if not 200/201."""
    if response.status_code not in (200, 201):
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
        f"{base_url}/signup", json=signup_payload, timeout=TIMEOUT)
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


def create_course(base_url: str, token: str, title: str, description: str) -> Dict[str, Any]:
    log_step(f"Creating course: {title}")
    response = requests.post(
        f"{base_url}/courses",
        json={"title": title, "description": description},
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


def setup_test_data(base_url: str, token: str) -> tuple[str, List[Dict[str, Any]], List[str]]:
    """Set up test data: create decks, course, and lessons."""
    log_step("Creating 4 decks...")
    decks: List[Dict[str, Any]] = []
    for i in range(4):
        deck = create_deck(base_url, token, f"deck-{i}")
        decks.append(deck)

    log_step("")
    course = create_course(base_url, token, "course-1", "course desc")
    course_id = course["id"]
    log_step(f"✓ Course created: {course_id}")

    log_step("")
    log_step("Creating 5 lessons...")
    lessons: List[Dict[str, Any]] = []
    lesson_deck_ids: List[str] = []
    for i in range(5):
        deck_id = decks[i % len(decks)]["id"] if i < 4 else decks[0]["id"]
        lesson = create_lesson(base_url, token, course_id,
                               order=i + 1, title=f"lesson-{i}", deck_id=deck_id)
        lessons.append(lesson)
        lesson_deck_ids.append(deck_id)

    return course_id, lessons, lesson_deck_ids


def run_pagination_before_answering(base_url: str, token: str, course_id: str, lessons: List[Dict[str, Any]]) -> None:
    """Test forward and backward pagination before answering."""
    log_step("")
    log_step("Testing forward pagination...")
    pages, edges = paginate_forward(base_url, token, course_id)
    log_step(
        f"✓ Forward pagination: {len(pages)} page(s), {len(edges)} edge(s)")
    assert len(edges) >= len(lessons)

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
        (e["lesson"] for e in focused_edges if e["lesson"]["order"] == 1), None)
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
        last_lesson = focused_all_edges[-1]["lesson"]
        assert last_lesson.get(
            "is_current") is True, f"Last lesson is_current should be True after completion, got {last_lesson.get('is_current')}"
        log_step(
            f"✓ Last lesson (order {last_lesson['order']}) has is_current=True")

        # Check all lessons are complete
        for edge in focused_all_edges:
            lesson_data = edge["lesson"]
            assert lesson_data.get(
                "is_completed") is True, f"Lesson {lesson_data['id']} should be completed"
        log_step(
            f"✓ All {len(focused_all_edges)} focused lessons are marked complete")


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


def test_basic_flow():
    init_log_file()

    base_url = load_base_url()
    log_step(f"Starting test against {base_url}")

    user = create_user(base_url)
    log_step(f"✓ User created: {user['username']}")
    token = user["token"]

    log_step("")
    course_id, lessons, lesson_deck_ids = setup_test_data(base_url, token)

    log_step("")
    enroll_course(base_url, token, course_id)

    run_pagination_before_answering(base_url, token, course_id, lessons)

    run_focused_lessons_before_answering(base_url, token, course_id)

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

    log_step("")
    log_step("✅ Test passed!")
    log_step(f"📝 Detailed logs written to: {LOG_FILE}")

    # Write HAR file
    write_har_file()
    log_step(f"📊 HAR file written to: {HAR_FILE}")
