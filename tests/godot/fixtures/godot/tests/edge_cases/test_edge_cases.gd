extends VestTest

func test_field_numbers_at_wire_format_boundaries():
	var msg := EdgeCasesLargeFieldNumbers.new()
	msg.set_small(1)
	msg.set_boundary(16)
	msg.set_mid_range(2046)
	msg.set_maximum(536870911)

	var bytes := msg.to_bytes()
	var decoded := EdgeCasesLargeFieldNumbers.new()
	expect_equal(decoded.from_bytes(bytes), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_small(), 1)
	expect_equal(decoded.get_boundary(), 16)
	expect_equal(decoded.get_mid_range(), 2046)
	expect_equal(decoded.get_maximum(), 536870911)

func test_reserved_fields_do_not_break_round_trip():
	var msg := EdgeCasesReservations.new()
	msg.set_active(7)
	msg.set_label("ok")

	var bytes := msg.to_bytes()
	var decoded := EdgeCasesReservations.new()
	expect_equal(decoded.from_bytes(bytes), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_active(), 7)
	expect_equal(decoded.get_label(), "ok")

func test_package_qualified_reference_round_trip():
	var holder := EdgeCasesPackageQualifiedReferences.new()
	var inner := holder.new_absolute_ref()
	inner.set_boundary(16)
	inner.set_maximum(536870911)

	var bytes := holder.to_bytes()
	var decoded := EdgeCasesPackageQualifiedReferences.new()
	expect_equal(decoded.from_bytes(bytes), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_absolute_ref().get_boundary(), 16)
	expect_equal(decoded.get_absolute_ref().get_maximum(), 536870911)

func test_field_numbers_round_trip_text():
	var msg := EdgeCasesLargeFieldNumbers.new()
	msg.set_small(3)
	msg.set_boundary(17)
	msg.set_mid_range(2046)
	msg.set_maximum(536870911)

	var text := msg.to_text()
	var decoded := EdgeCasesLargeFieldNumbers.new()
	expect_equal(decoded.from_text(text), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_small(), 3)
	expect_equal(decoded.get_boundary(), 17)
	expect_equal(decoded.get_mid_range(), 2046)
	expect_equal(decoded.get_maximum(), 536870911)
