extends VestTest

const CollectionsProto = preload("res://generated/collections.pb.gd")
const Core = preload("res://generated/proto_core_utils.gd")

func test_repeated_scalars_round_trip():
	var bag := CollectionsProto.Bag.new()
	bag.add_numbers(10)
	bag.add_numbers(20)
	bag.add_numbers(30)
	bag.add_labels("one")
	bag.add_labels("two")

	var bytes := bag.to_bytes()
	var decoded := CollectionsProto.Bag.new()
	expect_equal(decoded.from_bytes(bytes), Core.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_numbers(), [10, 20, 30] as Array[int])
	expect_equal(decoded.get_labels(), ["one", "two"] as Array[String])

func test_repeated_messages_round_trip():
	var bag := CollectionsProto.Bag.new()
	var first := bag.add_items()
	first.set_id(1)
	first.set_name("first")
	var second := bag.add_items()
	second.set_id(2)
	second.set_name("second")

	var bytes := bag.to_bytes()
	var decoded := CollectionsProto.Bag.new()
	expect_equal(decoded.from_bytes(bytes), Core.ProtobufError.NO_ERRORS)
	var got := decoded.get_items()
	expect_equal(got.size(), 2)
	expect_equal(got[0].get_id(), 1)
	expect_equal(got[0].get_name(), "first")
	expect_equal(got[1].get_id(), 2)
	expect_equal(got[1].get_name(), "second")

func test_scalar_keyed_maps_round_trip():
	var bag := CollectionsProto.Bag.new()
	bag.set_string_to_int("a", 1)
	bag.set_string_to_int("b", 2)
	bag.set_int_to_string(7, "seven")
	bag.set_int64_to_double(10, 1.5)
	bag.set_int64_to_double(20, 2.5)
	bag.set_bool_to_string(true, "yes")
	bag.set_bool_to_string(false, "no")

	var bytes := bag.to_bytes()
	var decoded := CollectionsProto.Bag.new()
	expect_equal(decoded.from_bytes(bytes), Core.ProtobufError.NO_ERRORS)

	var s2i := decoded.get_string_to_int()
	expect_equal(s2i["a"], 1)
	expect_equal(s2i["b"], 2)

	var i2s := decoded.get_int_to_string()
	expect_equal(i2s[7], "seven")

	var i64d := decoded.get_int64_to_double()
	expect_equal(i64d[10], 1.5)
	expect_equal(i64d[20], 2.5)

	var b2s := decoded.get_bool_to_string()
	expect_equal(b2s[true], "yes")
	expect_equal(b2s[false], "no")

func test_message_valued_map_round_trip():
	var bag := CollectionsProto.Bag.new()
	var item := CollectionsProto.ItemEntry.new()
	item.set_id(99)
	item.set_name("named")
	bag.set_string_to_message("k", item)

	var bytes := bag.to_bytes()
	var decoded := CollectionsProto.Bag.new()
	expect_equal(decoded.from_bytes(bytes), Core.ProtobufError.NO_ERRORS)
	var m := decoded.get_string_to_message()
	expect_equal(m["k"].get_id(), 99)
	expect_equal(m["k"].get_name(), "named")
