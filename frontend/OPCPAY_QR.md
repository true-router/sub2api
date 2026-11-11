# Direct OPCPay payment QR

The checkout QR encodes the validated `purchase_url`. The mobile payment link
uses the same URL. Treat `purchase_qr_code` as an optional upstream QR address;
encoding that address can make the first scan open another QR image.

This follows the [official OPCPay Node example](https://github.com/Yapie0/opcpay/blob/main/examples/node/server.mjs),
whose purchase QR endpoint encodes `value.purchase_url` directly. The
[official OpenAPI](https://ai6666.com/opcpay/api/v1/pay/developer/openapi.json)
defines `purchase_url` as the validated purchase entry.

The existing URL validation continues to permit HTTPS and Alipay app entries
without embedded credentials. This change does not create another purchase,
change the selected SKU or business ID, or confirm payment or credit from a scan.
Credit still requires the existing confirmed-consumption flow.

## End-to-end acceptance

Run from the repository root with the existing frontend dependencies and
ego-browser installed:

```sh
node frontend/e2e/opcpay-qr.mjs
```

The fixture starts an isolated API and the actual Vue checkout on port 13042.
The browser's native `BarcodeDetector` decodes the rendered QR image. Assertions
check the decoded target, rather than only checking that an image exists.
The browser must support the `qr_code` barcode format.

Scenarios cover upstream QR image addresses, rendering endpoints, missing QR
addresses, unsafe QR addresses, purchase recovery with the same business ID,
and the mobile Alipay link. The fixture never submits a consumption request or
performs a real payment. Fixture services stop when the command exits.

Artifacts are written to `output/opcpay-qr/after/`: `checks.json` includes source
hashes and decoded URLs, `requests.json` records fixture calls, and screenshots
show the desktop and mobile checkout. To capture the original failure before
applying the fix, run `node frontend/e2e/opcpay-qr.mjs before`; its artifacts are
kept separately in `output/opcpay-qr/before/`.

These checks validate the QR target and local checkout behavior. They do not
establish real-device Alipay navigation or merchant payment acceptance.
