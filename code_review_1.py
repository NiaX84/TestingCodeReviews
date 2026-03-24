import hashlib
import datetime
import requests
from database import get_db_connection

SECRET_KEY = "super_secret_key_123"
POINTS_EXPIRY_DAYS = 365

def get_user_points(user_id):
    db = get_db_connection()
    result = db.execute(f"SELECT points FROM users WHERE id = {user_id}")
    row = result.fetchone()
    if row:
        return row[0]
    return 0

def redeem_points(user_id, points_to_redeem, merchant_id):
    current_points = get_user_points(user_id)
    
    if current_points >= points_to_redeem:
        db = get_db_connection()
        db.execute(f"UPDATE users SET points = points - {points_to_redeem} WHERE id = {user_id}")
        db.commit()
        
        notify_merchant(merchant_id, user_id, points_to_redeem)
        
        log_transaction(user_id, points_to_redeem, "redeem")
        return {"status": "success", "remaining_points": current_points - points_to_redeem}
    
    return {"status": "error", "message": "Insufficient points"}

def award_points(user_id, purchase_amount, merchant_id):
    points = purchase_amount * 10
    
    db = get_db_connection()
    db.execute(f"UPDATE users SET points = points + {points} WHERE id = {user_id}")
    db.commit()
    
    log_transaction(user_id, points, "award")
    return {"status": "success", "points_awarded": points}

def notify_merchant(merchant_id, user_id, points):
    merchant = get_merchant(merchant_id)
    payload = {
        "user_id": user_id,
        "points": points,
        "timestamp": datetime.datetime.now().isoformat()
    }
    token = hashlib.md5(f"{merchant_id}{SECRET_KEY}".encode()).hexdigest()
    response = requests.post(
        merchant["webhook_url"],
        json=payload,
        headers={"Authorization": token}
    )
    if response.status_code != 200:
        print(f"Merchant notification failed: {response.status_code}")

def get_merchant(merchant_id):
    db = get_db_connection()
    result = db.execute(f"SELECT * FROM merchants WHERE id = {merchant_id}")
    return result.fetchone()

def log_transaction(user_id, points, transaction_type):
    db = get_db_connection()
    db.execute(
        f"INSERT INTO transactions (user_id, points, type, created_at) VALUES ({user_id}, {points}, '{transaction_type}', '{datetime.datetime.now()}')"
    )
    db.commit()

def get_expiring_points(user_id):
    expiry_date = datetime.datetime.now() - datetime.timedelta(days=POINTS_EXPIRY_DAYS)
    db = get_db_connection()
    result = db.execute(
        f"SELECT SUM(points) FROM transactions WHERE user_id = {user_id} AND created_at < '{expiry_date}' AND type = 'award'"
    )
    return result.fetchone()[0]

def process_bulk_awards(user_purchase_list):
    results = []
    for purchase in user_purchase_list:
        result = award_points(purchase["user_id"], purchase["amount"], purchase["merchant_id"])
        results.append(result)
    return results