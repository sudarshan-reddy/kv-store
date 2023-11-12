import pytest
import requests

BASE_URL_MAP_WITH_MUTEX = "http://localhost:11200"
BASE_URL_LRU = "http://localhost:11201"

@pytest.mark.parametrize("base_url", [BASE_URL_MAP_WITH_MUTEX, BASE_URL_LRU])
def test_set_and_get(base_url):
    key, value = "testKey", "testValue"
    
    # Test Set
    set_response = requests.post(f"{base_url}/set", json={"key": key, "value": value})
    assert set_response.status_code == 201

    # Test Get
    get_response = requests.get(f"{base_url}/get", params={"key": key})
    assert get_response.status_code == 200
    assert get_response.json()['value'] == value

@pytest.mark.parametrize("base_url", [BASE_URL_MAP_WITH_MUTEX, BASE_URL_LRU])
def test_update_bulk_full_success(base_url):
    pairs = [{"key": f"key{i}", "value": f"value{i}"} for i in range(5)]

    # Prepopulate keys
    for pair in pairs:
        requests.post(f"{base_url}/set", json=pair)

    # Test Bulk Update
    update_response = requests.patch(f"{base_url}/updateBulk", json=pairs)
    assert update_response.status_code == 200

    # Verify updates
    for pair in pairs:
        get_response = requests.get(f"{base_url}/get", params={"key": pair["key"]})
        assert get_response.status_code == 200
        assert get_response.json()['value'] == pair["value"]

@pytest.mark.parametrize("base_url", [BASE_URL_MAP_WITH_MUTEX, BASE_URL_LRU])
def test_update_bulk_partial_success(base_url):
    existing_pairs = [{"key": f"existing{i}", "value": f"value{i}"} for i in range(3)]
    non_existing_pairs = [{"key": f"nonexisting{i}", "value": f"value{i}"} for i in range(3, 5)]

    # Prepopulate some keys
    for pair in existing_pairs:
        requests.post(f"{base_url}/set", json=pair)

    # Test Bulk Update with some non-existing keys
    update_response = requests.patch(f"{base_url}/updateBulk", json=existing_pairs + non_existing_pairs)
    assert update_response.status_code == 206

    # Verify updates for existing keys
    for pair in existing_pairs:
        get_response = requests.get(f"{base_url}/get", params={"key": pair["key"]})
        assert get_response.status_code == 200
        assert get_response.json()['value'] == pair["value"]

    # Verify non-existing keys were not updated
    for pair in non_existing_pairs:
        get_response = requests.get(f"{base_url}/get", params={"key": pair["key"]})
        assert get_response.status_code == 404

@pytest.mark.parametrize("base_url", [BASE_URL_MAP_WITH_MUTEX, BASE_URL_LRU])
def test_delete(base_url):
    key = "deleteKey"
    
    # Setup - ensure key exists
    requests.post(f"{base_url}/set", json={"key": key, "value": "toDelete"})

    # Test Delete
    delete_response = requests.delete(f"{base_url}/delete", params={"key": key})
    assert delete_response.status_code == 200

    # Verify deletion
    get_response = requests.get(f"{base_url}/get", params={"key": key})
    assert get_response.status_code == 404

