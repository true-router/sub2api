# Customer-facing recharge confirmation

The checkout shows the payment currency and price, and the USD credit delivered
after confirmation. Provider points, entitlement balances and redemption details
are internal accounting concepts and are not displayed in the checkout.
Merchant SKU names may contain internal units, so cards display the payment
amount and credited USD amount instead of using those names as customer labels.

## Definitive verification failure

The server returns HTTP 409 with reason `OPCPAY_PAYMENT_UNVERIFIED` and the
customer message `验证失败，请重试` when access verification rejects the request
before creating or submitting a consumption operation.

Only that structured reason restores the saved action to its purchase phase.
The original action ID, SKU and payment entry remain available for retry.
Closing the dialog then closes the purchase intent and permits a new checkout.
A previously stuck session retries its original action before restoring the
purchase dialog after the same definitive rejection.

Quantity access does not require the buyout-only `decision` field to be present.
An explicit denial still rejects access. Granted access, consumption capability,
the expected unit and sufficient remaining quantity are all required.

## Uncertain results

A timeout, generic conflict, unknown response or confirmed consumption awaiting
delivery retains its consumption phase and original action ID. A pending purchase
in the next information response is not sufficient evidence to abandon it.
If the dialog is closed, the user can query the original recharge result.
Confirmed consumption restores delivery through the existing idempotent path.

## Repeatable acceptance

From the repository root:

```sh
node frontend/e2e/opcpay-confirmation.mjs
GOTOOLCHAIN=go1.27.4 backend/tools/opcpay-e2e/run.sh
```

The browser fixture exercises the real Vue checkout with a synthetic session
and provider. It checks payment/credit wording, unpaid confirmation and retry,
stuck-session recovery, closing and new checkout, unknown responses, result
queries and confirmed-delivery recovery. Results, source hashes and a screenshot
are written to `output/opcpay-confirmation/after/`. Use the `before` argument to
capture the pre-fix failure in a separate directory.

The backend fixture uses real HTTP handlers and a disposable PostgreSQL database.
It verifies the structured rejection, absence of a consumption row or credit on
rejection, cancellation and a new purchase, and existing recovery, concurrency,
delivery, provider-binding and credential-protection behavior. Its receipt is
`output/opcpay-e2e/checks.json`.

These fixtures do not perform real payments or consume real customer funds.
