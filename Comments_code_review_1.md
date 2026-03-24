The areas to focus on during review:

1. Naming clarity and intent
 * we have a static method library that deals with points awarded to user within a merchant
 * I would suggest instead of static method library this to be a class, that handles user points within a merchant. 
 * There is no connection between merchant and user when dealing with points. It just adds points without dealing with the question which merchant the user is currently in
 * instead of using direct queries, a wrapper around the DB model woudl be better, like having a Django solution for example with models and views
 * get_user_points method 
   * does not specify merchant
 * transaction_type should be defined outside as a constant/enum/dictionary
 * unused merchant_id in award points
 * knowledge of structures that are hidden is necessary in order to review correctness of lines like merchant["webhook_url"]

2. Error handling gaps
 * there is almost no error handling
 * missing error handlings:
   * getting db connection - does the method return success all the time? What if the connection fails? Having a try/catch block would imprve that
 * method redeem_points:
   * notify_merchant can fail, no error handling is present

3. Missing Edge cases
 * fetchone() - no check on result that we are expecting exactly one user_id
 * get_merchant - result - missing handling of cases when 0 or multiple rows are returned.
 * merchant is not notified of all transactions
 * missing db.commit()

4. Testability/coupling
 * get_db_connection - possiblity of mocking this or using in a different environment is missing

5. Performance red flags (N+1), unbounded loops)
 * multiple get_db_connection calls, this can exhaust resources on DB
 * no DB connection close
 * process_bulk_awards - unperformant for loop. Can use map function

6. Security/Input Validation
 * security key is stored in file and is the same for all environments and merchants
 * no input validation