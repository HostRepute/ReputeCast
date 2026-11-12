from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.conversion_item_matched import ConversionItemMatched
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.conversion_credit import ConversionCredit





T = TypeVar("T", bound="ConversionItem")



@_attrs_define
class ConversionItem:
    """ 
        Attributes:
            currency (str):
            id (str):
            key_name (str):
            matched (ConversionItemMatched):
            name (str):
            occurred_at (datetime.datetime):
            received_at (datetime.datetime):
            test (bool):
            value (str):
            value_cents (int):
            credited (ConversionCredit | Unset):
     """

    currency: str
    id: str
    key_name: str
    matched: ConversionItemMatched
    name: str
    occurred_at: datetime.datetime
    received_at: datetime.datetime
    test: bool
    value: str
    value_cents: int
    credited: ConversionCredit | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.conversion_credit import ConversionCredit # noqa: PLC0415
        currency = self.currency

        id = self.id

        key_name = self.key_name

        matched = self.matched.value

        name = self.name

        occurred_at = self.occurred_at.isoformat()

        received_at = self.received_at.isoformat()

        test = self.test

        value = self.value

        value_cents = self.value_cents

        credited: dict[str, Any] | Unset = UNSET
        if not isinstance(self.credited, Unset):
            credited = self.credited.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "currency": currency,
            "id": id,
            "keyName": key_name,
            "matched": matched,
            "name": name,
            "occurredAt": occurred_at,
            "receivedAt": received_at,
            "test": test,
            "value": value,
            "valueCents": value_cents,
        })
        if credited is not UNSET:
            field_dict["credited"] = credited

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.conversion_credit import ConversionCredit # noqa: PLC0415
        d = dict(src_dict)
        currency = d.pop("currency")

        id = d.pop("id")

        key_name = d.pop("keyName")

        matched = ConversionItemMatched(d.pop("matched"))




        name = d.pop("name")

        occurred_at = datetime.datetime.fromisoformat(d.pop("occurredAt"))




        received_at = datetime.datetime.fromisoformat(d.pop("receivedAt"))




        test = d.pop("test")

        value = d.pop("value")

        value_cents = d.pop("valueCents")

        _credited = d.pop("credited", UNSET)
        credited: ConversionCredit | Unset
        if isinstance(_credited,  Unset):
            credited = UNSET
        else:
            credited = ConversionCredit.from_dict(_credited)




        conversion_item = cls(
            currency=currency,
            id=id,
            key_name=key_name,
            matched=matched,
            name=name,
            occurred_at=occurred_at,
            received_at=received_at,
            test=test,
            value=value,
            value_cents=value_cents,
            credited=credited,
        )

        return conversion_item

