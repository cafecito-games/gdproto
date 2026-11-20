extends VestTest

const SAVE_PATH := "user://gdproto-no-text-resource.tres"


func test_text_format_methods_omitted_and_debug_string_preserved():
	var empty := MessagesEmptyMessage.new()
	expect_equal(empty.has_method("to_text"), false)
	expect_equal(empty.has_method("from_text"), false)
	expect_equal(empty.has_method("to_bytes"), true)
	expect_equal(empty.has_method("from_bytes"), true)
	expect_equal(str(empty), "EmptyMessage {}")

	var host := MessagesNestedHost.new()
	expect_equal(host.has_method("to_text"), false)
	expect_equal(host.has_method("from_text"), false)
	host.new_outer().new_middle().new_inner().set_value(123)
	host.new_empty()

	var decoded_host := MessagesNestedHost.new()
	expect_equal(decoded_host.from_bytes(host.to_bytes()), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded_host.get_outer().get_middle().get_inner().get_value(), 123)
	expect_equal(str(decoded_host).contains("outer:"), true)


func test_imported_messages_and_enums_binary_round_trip_without_text_format():
	var event := EventEvent.new()
	expect_equal(event.has_method("to_text"), false)
	expect_equal(event.has_method("from_text"), false)
	event.new_occurred_at().set_seconds(1700000000)
	event.set_color(SharedHue.Hue.BLUE)
	event.new_payload().set_data("binary-only")
	event.set_kind(SharedSourceKind.SourceKind.SYSTEM)

	var decoded := EventEvent.new()
	expect_equal(decoded.from_bytes(event.to_bytes()), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_occurred_at().get_seconds(), 1700000000)
	expect_equal(decoded.get_color(), SharedHue.Hue.BLUE)
	expect_equal(decoded.get_payload().get_data(), "binary-only")
	expect_equal(decoded.get_source_case(), EventEvent.SourceOneOf.KIND)
	expect_equal(decoded.get_kind(), SharedSourceKind.SourceKind.SYSTEM)
	expect_equal(str(decoded).contains("Event {"), true)


func test_oneof_optional_collections_and_resource_without_text_format():
	var oneof_msg := OneofOneofHost.new()
	expect_equal(oneof_msg.has_method("to_text"), false)
	expect_equal(oneof_msg.has_method("from_text"), false)
	oneof_msg.set_number(5)
	oneof_msg.set_text("hello")
	expect_equal(oneof_msg.get_number(), 0)
	expect_equal(oneof_msg.get_text(), "hello")
	var decoded_oneof := OneofOneofHost.new()
	expect_equal(decoded_oneof.from_bytes(oneof_msg.to_bytes()), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded_oneof.get_choice_case(), OneofOneofHost.ChoiceOneOf.TEXT)
	expect_equal(decoded_oneof.get_text(), "hello")

	var opt := OptionalOptionalScalars.new()
	expect_equal(opt.has_method("to_text"), false)
	opt.set_int_value(0)
	var decoded_opt := OptionalOptionalScalars.new()
	expect_equal(decoded_opt.from_bytes(opt.to_bytes()), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded_opt.has_int_value(), true)
	expect_equal(decoded_opt.get_int_value(), 0)

	var bag := CollectionsBag.new()
	expect_equal(bag.has_method("to_text"), false)
	bag.add_numbers(10)
	bag.add_labels("tag")
	bag.add_string_to_int("hp", 50)
	var decoded_bag := CollectionsBag.new()
	expect_equal(decoded_bag.from_bytes(bag.to_bytes()), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded_bag.get_numbers(), [10] as Array[int])
	expect_equal(decoded_bag.get_labels(), ["tag"] as Array[String])
	expect_equal(decoded_bag.get_string_to_int()["hp"], 50)

	_remove_saved_resource()
	var stored := ResourceStoredMessage.new()
	expect_equal(stored.has_method("to_text"), false)
	expect_equal(stored.has_method("from_text"), false)
	stored.set_name_pb("wire-only")
	stored.set_level(0)
	stored.set_count(9)
	expect_equal(ResourceSaver.save(stored, SAVE_PATH), OK)
	var loaded := ResourceLoader.load(
		SAVE_PATH,
		"",
		ResourceLoader.CACHE_MODE_IGNORE,
	) as ResourceStoredMessage
	expect_equal(loaded == null, false)
	expect_equal(loaded.get_name_pb(), "wire-only")
	expect_equal(loaded.has_level(), true)
	expect_equal(loaded.get_count(), 9)
	expect_equal(loaded.to_bytes(), stored.to_bytes())
	expect_equal(str(loaded).contains("StoredMessage {"), true)
	_remove_saved_resource()


func _remove_saved_resource():
	var absolute_path := ProjectSettings.globalize_path(SAVE_PATH)
	if FileAccess.file_exists(SAVE_PATH):
		DirAccess.remove_absolute(absolute_path)
