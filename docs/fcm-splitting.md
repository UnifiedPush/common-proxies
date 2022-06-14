# FCM rewrite proxy payload splitting

UnifiedPush requests can be up to 4096 bytes, when converted to base64, that becomes 5464 bytes (x4/3 + padding). FCM's limit is 4096 (including other metadata, so actually less than 4096).

FCM Splitting basically uses the following logic:

```py
assert len(payload) <= 4096

b64 = b64encode(payload)
if len(b64) > 3800: # 3800 is a rough amount that determines how much can easily fit in an FCM request
	split(b64)
else: #if it's short enough
	send({"b": b64, "i": instance})

# split works like the following
def split(b64):
  rand_num = randomstring(20) # up to 20 character random string
  send({"b": b64[:3000], "m": m, "s": "1", "i": instance})
  send({"b": b64[3000:], "m": m, "s": "2", "i": instance})
```

The first 3000 base64'd bytes get sent out in one push notification, and anything after 3000 is sent out in another. Both are linked using the random id "m", which is less than or equal to 20 characters long. In practice, "m" is an integer, but it MUST be treated as a string, because its integer-ness doesn't mean anything.
Then "s" "1" and "2" are used to identify identify the order of the chunks.

See an example decoding implementation here: <https://github.com/UnifiedPush/android-foss_embedded_fcm_distributor/commit/812f0f6f3badd5870bfe7d8b317f80942f741458>
