extends VestTest

const SAVE_PATH := "user://gdproto-resource-round-trip.tres"


func test_tres_round_trip_preserves_protobuf_state():
	_remove_saved_resource()
	var original := ResourceStoredMessage.new()
	expect_equal(original is Resource, true)
	original.set_name_pb("stored")
	original.set_level(0)
	original.set_payload(PackedByteArray([1, 2, 3]))
	original.set_mode(ResourceResourceMode.ResourceMode.RESOURCE_MODE_ACTIVE)
	original.add_tags("one")
	original.add_tags("two")
	original.add_attributes("attack", 12)
	original.new_ref_counted_child().set_value(7)
	original.new_resource_child().set_value(8)
	original.set_count(0)

	var expected_bytes := original.to_bytes()
	var expected_text := original.to_text()
	expect_equal(ResourceSaver.save(original, SAVE_PATH), OK)

	var loaded_resource := ResourceLoader.load(
		SAVE_PATH,
		"",
		ResourceLoader.CACHE_MODE_IGNORE,
	)
	var loaded := loaded_resource as ResourceStoredMessage
	expect_equal(loaded == null, false)
	expect_equal(loaded == original, false)
	expect_equal(loaded.get_name_pb(), "stored")
	expect_equal(loaded.has_level(), true)
	expect_equal(loaded.get_level(), 0)
	expect_equal(loaded.get_payload(), PackedByteArray([1, 2, 3]))
	expect_equal(loaded.get_mode(), ResourceResourceMode.ResourceMode.RESOURCE_MODE_ACTIVE)
	expect_equal(loaded.get_tags(), ["one", "two"] as Array[String])
	expect_equal(loaded.get_attributes()["attack"], 12)
	expect_equal(loaded.get_ref_counted_child().get_value(), 7)
	var ref_counted_child: Variant = loaded.get_ref_counted_child()
	expect_equal(ref_counted_child is Resource, false)
	expect_equal(loaded.get_resource_child().get_value(), 8)
	expect_equal(loaded.get_resource_child() is Resource, true)
	expect_equal(loaded.get_choice_case(), ResourceStoredMessage.ChoiceOneOf.COUNT)
	expect_equal(loaded.get_count(), 0)
	expect_equal(loaded.to_bytes(), expected_bytes)
	expect_equal(loaded.to_text(), expected_text)
	_remove_saved_resource()


func _remove_saved_resource():
	var absolute_path := ProjectSettings.globalize_path(SAVE_PATH)
	if FileAccess.file_exists(SAVE_PATH):
		DirAccess.remove_absolute(absolute_path)
