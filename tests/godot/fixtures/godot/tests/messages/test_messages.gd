extends VestTest

func test_empty_message_round_trip():
	var msg := MessagesProto.EmptyMessage.new()
	var bytes := msg.to_bytes()
	expect_equal(bytes.size(), 0)

	var decoded := MessagesProto.EmptyMessage.new()
	expect_equal(decoded.from_bytes(bytes), ProtoCoreUtils.ProtobufError.NO_ERRORS)

func test_deeply_nested_round_trip():
	var host := MessagesProto.NestedHost.new()
	var outer := host.new_outer()
	var middle := outer.new_middle()
	var inner := middle.new_inner()
	inner.set_value(42)

	var bytes := host.to_bytes()
	var decoded := MessagesProto.NestedHost.new()
	expect_equal(decoded.from_bytes(bytes), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_outer().get_middle().get_inner().get_value(), 42)

func test_self_reference_round_trip():
	var head := MessagesProto.LinkedListNode.new()
	head.set_value(1)
	var second := head.new_next()
	second.set_value(2)
	var third := second.new_next()
	third.set_value(3)

	var bytes := head.to_bytes()
	var decoded := MessagesProto.LinkedListNode.new()
	expect_equal(decoded.from_bytes(bytes), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_value(), 1)
	expect_equal(decoded.get_next().get_value(), 2)
	expect_equal(decoded.get_next().get_next().get_value(), 3)

func test_deeply_nested_round_trip_text():
	var host := MessagesProto.NestedHost.new()
	var outer := host.new_outer()
	var middle := outer.new_middle()
	var inner := middle.new_inner()
	inner.set_value(7)

	var text := host.to_text()
	var decoded := MessagesProto.NestedHost.new()
	expect_equal(decoded.from_text(text), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_outer().get_middle().get_inner().get_value(), 7)
