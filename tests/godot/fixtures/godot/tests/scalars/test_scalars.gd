extends VestTest

const ScalarsProto = preload("res://generated/scalars.pb.gd")
const Core = preload("res://generated/proto_core_utils.gd")

func _populate(msg: ScalarsProto.AllScalars) -> void:
	msg.set_v_int32(-2147483648)
	msg.set_v_int64(-9223372036854775808)
	msg.set_v_uint32(4294967295)
	msg.set_v_uint64(9223372036854775807) # GDScript int is signed 64-bit; uint64 max won't fit.
	msg.set_v_sint32(-1)
	msg.set_v_sint64(-2)
	msg.set_v_fixed32(123456)
	msg.set_v_fixed64(7890123)
	msg.set_v_sfixed32(-321)
	msg.set_v_sfixed64(-987)
	msg.set_v_float(1.5)
	msg.set_v_double(3.141592653589793)
	msg.set_v_bool(true)
	msg.set_v_string("héllo, 世界")
	msg.set_v_bytes(PackedByteArray([0, 1, 2, 255]))
	msg.add_packed_ints(1)
	msg.add_packed_ints(2)
	msg.add_packed_ints(3)

func _expect_populated(msg: ScalarsProto.AllScalars) -> void:
	expect_equal(msg.get_v_int32(), -2147483648)
	expect_equal(msg.get_v_int64(), -9223372036854775808)
	expect_equal(msg.get_v_uint32(), 4294967295)
	expect_equal(msg.get_v_uint64(), 9223372036854775807)
	expect_equal(msg.get_v_sint32(), -1)
	expect_equal(msg.get_v_sint64(), -2)
	expect_equal(msg.get_v_fixed32(), 123456)
	expect_equal(msg.get_v_fixed64(), 7890123)
	expect_equal(msg.get_v_sfixed32(), -321)
	expect_equal(msg.get_v_sfixed64(), -987)
	expect_equal(msg.get_v_float(), 1.5)
	expect_equal(msg.get_v_double(), 3.141592653589793)
	expect_equal(msg.get_v_bool(), true)
	expect_equal(msg.get_v_string(), "héllo, 世界")
	expect_equal(msg.get_v_bytes(), PackedByteArray([0, 1, 2, 255]))
	expect_equal(msg.get_packed_ints(), [1, 2, 3] as Array[int])

func test_default_values_round_trip_binary():
	var msg := ScalarsProto.AllScalars.new()
	var bytes := msg.to_bytes()
	# proto3 omits scalar defaults, so the wire form is empty.
	expect_equal(bytes.size(), 0)

	var decoded := ScalarsProto.AllScalars.new()
	expect_equal(decoded.from_bytes(bytes), Core.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_v_int32(), 0)
	expect_equal(decoded.get_v_string(), "")
	expect_equal(decoded.get_v_bool(), false)
	expect_equal(decoded.get_v_bytes(), PackedByteArray())

func test_extreme_values_round_trip_binary():
	var msg := ScalarsProto.AllScalars.new()
	_populate(msg)
	var bytes := msg.to_bytes()

	var decoded := ScalarsProto.AllScalars.new()
	expect_equal(decoded.from_bytes(bytes), Core.ProtobufError.NO_ERRORS)
	_expect_populated(decoded)

func test_extreme_values_round_trip_text():
	# Bytes go through to_utf8_buffer in from_text, so we use a UTF-8 payload
	# here. Non-UTF-8 byte sequences are still binary-only.
	var msg := ScalarsProto.AllScalars.new()
	_populate(msg)
	msg.set_v_bytes("ascii".to_utf8_buffer())
	var text := msg.to_text()

	var decoded := ScalarsProto.AllScalars.new()
	expect_equal(decoded.from_text(text), Core.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_v_int32(), -2147483648)
	expect_equal(decoded.get_v_string(), "héllo, 世界")
	expect_equal(decoded.get_v_bool(), true)
	expect_equal(decoded.get_packed_ints(), [1, 2, 3] as Array[int])
	expect_equal(decoded.get_v_bytes(), "ascii".to_utf8_buffer())
