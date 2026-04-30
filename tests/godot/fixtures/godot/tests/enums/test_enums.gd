extends VestTest

const EnumsProto = preload("res://generated/enums.pb.gd")
const Core = preload("res://generated/proto_core_utils.gd")

func test_top_level_and_nested_enum_round_trip():
	var msg := EnumsProto.EnumHost.new()
	msg.set_mode(EnumsProto.TopLevelMode.TOP_LEVEL_MODE_ALPHA)
	msg.set_nested(EnumsProto.EnumHost.Nested.NESTED_TWO)
	var bytes := msg.to_bytes()

	var decoded := EnumsProto.EnumHost.new()
	expect_equal(decoded.from_bytes(bytes), Core.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_mode(), EnumsProto.TopLevelMode.TOP_LEVEL_MODE_ALPHA)
	expect_equal(decoded.get_nested(), EnumsProto.EnumHost.Nested.NESTED_TWO)

func test_aliased_enum_resolves_to_shared_integer():
	var msg := EnumsProto.EnumHost.new()
	msg.set_rank(EnumsProto.AliasedRank.ALIASED_RANK_PRIMARY)
	# allow_alias means GOLD == PRIMARY; both must compare equal.
	expect_equal(msg.get_rank(), EnumsProto.AliasedRank.ALIASED_RANK_GOLD)

	var bytes := msg.to_bytes()
	var decoded := EnumsProto.EnumHost.new()
	expect_equal(decoded.from_bytes(bytes), Core.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_rank(), EnumsProto.AliasedRank.ALIASED_RANK_GOLD)

func test_repeated_enum_round_trip():
	var msg := EnumsProto.EnumHost.new()
	msg.add_mode_history(EnumsProto.TopLevelMode.TOP_LEVEL_MODE_ALPHA)
	msg.add_mode_history(EnumsProto.TopLevelMode.TOP_LEVEL_MODE_BETA)
	msg.add_mode_history(EnumsProto.TopLevelMode.TOP_LEVEL_MODE_UNSPECIFIED)
	var bytes := msg.to_bytes()

	var decoded := EnumsProto.EnumHost.new()
	expect_equal(decoded.from_bytes(bytes), Core.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_mode_history(), [
		EnumsProto.TopLevelMode.TOP_LEVEL_MODE_ALPHA,
		EnumsProto.TopLevelMode.TOP_LEVEL_MODE_BETA,
		EnumsProto.TopLevelMode.TOP_LEVEL_MODE_UNSPECIFIED,
	] as Array[int])

func test_enum_valued_map_round_trip():
	var msg := EnumsProto.EnumHost.new()
	msg.set_mode_by_id(1, EnumsProto.TopLevelMode.TOP_LEVEL_MODE_ALPHA)
	msg.set_mode_by_id(2, EnumsProto.TopLevelMode.TOP_LEVEL_MODE_BETA)
	var bytes := msg.to_bytes()

	var decoded := EnumsProto.EnumHost.new()
	expect_equal(decoded.from_bytes(bytes), Core.ProtobufError.NO_ERRORS)
	var got := decoded.get_mode_by_id()
	expect_equal(got[1], EnumsProto.TopLevelMode.TOP_LEVEL_MODE_ALPHA)
	expect_equal(got[2], EnumsProto.TopLevelMode.TOP_LEVEL_MODE_BETA)

func test_enum_round_trip_text():
	var msg := EnumsProto.EnumHost.new()
	msg.set_mode(EnumsProto.TopLevelMode.TOP_LEVEL_MODE_BETA)
	msg.set_nested(EnumsProto.EnumHost.Nested.NESTED_ONE)
	var text := msg.to_text()

	var decoded := EnumsProto.EnumHost.new()
	expect_equal(decoded.from_text(text), Core.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_mode(), EnumsProto.TopLevelMode.TOP_LEVEL_MODE_BETA)
	expect_equal(decoded.get_nested(), EnumsProto.EnumHost.Nested.NESTED_ONE)
