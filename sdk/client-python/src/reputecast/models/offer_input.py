from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.offer_input_discount_type import OfferInputDiscountType
from ..models.offer_input_duration import OfferInputDuration
from ..models.offer_input_kind import OfferInputKind
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.offer_applies_input import OfferAppliesInput





T = TypeVar("T", bound="OfferInput")



@_attrs_define
class OfferInput:
    """ 
        Attributes:
            applies (list[OfferAppliesInput]):
            discount_type (OfferInputDiscountType):
            duration (OfferInputDuration):
            kind (OfferInputKind):
            name (str):
            amount_off_cents (int | Unset):
            by_code (bool | Unset):
            by_link (bool | Unset):
            code (str | Unset):
            code_days (int | Unset):
            ends_on (str | Unset): YYYY-MM-DD
            limit (int | Unset):
            min_paid_months (int | Unset):
            months (int | Unset):
            new_only (bool | Unset):
            percent_off (int | Unset):
     """

    applies: list[OfferAppliesInput]
    discount_type: OfferInputDiscountType
    duration: OfferInputDuration
    kind: OfferInputKind
    name: str
    amount_off_cents: int | Unset = UNSET
    by_code: bool | Unset = UNSET
    by_link: bool | Unset = UNSET
    code: str | Unset = UNSET
    code_days: int | Unset = UNSET
    ends_on: str | Unset = UNSET
    limit: int | Unset = UNSET
    min_paid_months: int | Unset = UNSET
    months: int | Unset = UNSET
    new_only: bool | Unset = UNSET
    percent_off: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.offer_applies_input import OfferAppliesInput # noqa: PLC0415
        applies = []
        for applies_item_data in self.applies:
            applies_item = applies_item_data.to_dict()
            applies.append(applies_item)



        discount_type = self.discount_type.value

        duration = self.duration.value

        kind = self.kind.value

        name = self.name

        amount_off_cents = self.amount_off_cents

        by_code = self.by_code

        by_link = self.by_link

        code = self.code

        code_days = self.code_days

        ends_on = self.ends_on

        limit = self.limit

        min_paid_months = self.min_paid_months

        months = self.months

        new_only = self.new_only

        percent_off = self.percent_off


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "applies": applies,
            "discountType": discount_type,
            "duration": duration,
            "kind": kind,
            "name": name,
        })
        if amount_off_cents is not UNSET:
            field_dict["amountOffCents"] = amount_off_cents
        if by_code is not UNSET:
            field_dict["byCode"] = by_code
        if by_link is not UNSET:
            field_dict["byLink"] = by_link
        if code is not UNSET:
            field_dict["code"] = code
        if code_days is not UNSET:
            field_dict["codeDays"] = code_days
        if ends_on is not UNSET:
            field_dict["endsOn"] = ends_on
        if limit is not UNSET:
            field_dict["limit"] = limit
        if min_paid_months is not UNSET:
            field_dict["minPaidMonths"] = min_paid_months
        if months is not UNSET:
            field_dict["months"] = months
        if new_only is not UNSET:
            field_dict["newOnly"] = new_only
        if percent_off is not UNSET:
            field_dict["percentOff"] = percent_off

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.offer_applies_input import OfferAppliesInput # noqa: PLC0415
        d = dict(src_dict)
        applies = []
        _applies = d.pop("applies")
        for applies_item_data in (_applies):
            applies_item = OfferAppliesInput.from_dict(applies_item_data)



            applies.append(applies_item)


        discount_type = OfferInputDiscountType(d.pop("discountType"))




        duration = OfferInputDuration(d.pop("duration"))




        kind = OfferInputKind(d.pop("kind"))




        name = d.pop("name")

        amount_off_cents = d.pop("amountOffCents", UNSET)

        by_code = d.pop("byCode", UNSET)

        by_link = d.pop("byLink", UNSET)

        code = d.pop("code", UNSET)

        code_days = d.pop("codeDays", UNSET)

        ends_on = d.pop("endsOn", UNSET)

        limit = d.pop("limit", UNSET)

        min_paid_months = d.pop("minPaidMonths", UNSET)

        months = d.pop("months", UNSET)

        new_only = d.pop("newOnly", UNSET)

        percent_off = d.pop("percentOff", UNSET)

        offer_input = cls(
            applies=applies,
            discount_type=discount_type,
            duration=duration,
            kind=kind,
            name=name,
            amount_off_cents=amount_off_cents,
            by_code=by_code,
            by_link=by_link,
            code=code,
            code_days=code_days,
            ends_on=ends_on,
            limit=limit,
            min_paid_months=min_paid_months,
            months=months,
            new_only=new_only,
            percent_off=percent_off,
        )

        return offer_input

