from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.whats_app_cost_country import WhatsAppCostCountry





T = TypeVar("T", bound="WhatsAppCost")



@_attrs_define
class WhatsAppCost:
    """ 
        Attributes:
            amount (float):
            by_country (list[WhatsAppCostCountry]):
            category (str):
            messages (int):
            rates_as_of (datetime.datetime):
     """

    amount: float
    by_country: list[WhatsAppCostCountry]
    category: str
    messages: int
    rates_as_of: datetime.datetime





    def to_dict(self) -> dict[str, Any]:
        from ..models.whats_app_cost_country import WhatsAppCostCountry # noqa: PLC0415
        amount = self.amount

        by_country = []
        for by_country_item_data in self.by_country:
            by_country_item = by_country_item_data.to_dict()
            by_country.append(by_country_item)



        category = self.category

        messages = self.messages

        rates_as_of = self.rates_as_of.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "amount": amount,
            "byCountry": by_country,
            "category": category,
            "messages": messages,
            "ratesAsOf": rates_as_of,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.whats_app_cost_country import WhatsAppCostCountry # noqa: PLC0415
        d = dict(src_dict)
        amount = d.pop("amount")

        by_country = []
        _by_country = d.pop("byCountry")
        for by_country_item_data in (_by_country):
            by_country_item = WhatsAppCostCountry.from_dict(by_country_item_data)



            by_country.append(by_country_item)


        category = d.pop("category")

        messages = d.pop("messages")

        rates_as_of = datetime.datetime.fromisoformat(d.pop("ratesAsOf"))




        whats_app_cost = cls(
            amount=amount,
            by_country=by_country,
            category=category,
            messages=messages,
            rates_as_of=rates_as_of,
        )

        return whats_app_cost

