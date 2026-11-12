from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="WebsiteTierOption")



@_attrs_define
class WebsiteTierOption:
    """ 
        Attributes:
            currency (str):
            id (str):
            label (str):
            monthly (int):
            name (str):
            public (bool):
            trial_days (int):
            yearly (int):
            includes (str | Unset):
     """

    currency: str
    id: str
    label: str
    monthly: int
    name: str
    public: bool
    trial_days: int
    yearly: int
    includes: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        currency = self.currency

        id = self.id

        label = self.label

        monthly = self.monthly

        name = self.name

        public = self.public

        trial_days = self.trial_days

        yearly = self.yearly

        includes = self.includes


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "currency": currency,
            "id": id,
            "label": label,
            "monthly": monthly,
            "name": name,
            "public": public,
            "trialDays": trial_days,
            "yearly": yearly,
        })
        if includes is not UNSET:
            field_dict["includes"] = includes

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        currency = d.pop("currency")

        id = d.pop("id")

        label = d.pop("label")

        monthly = d.pop("monthly")

        name = d.pop("name")

        public = d.pop("public")

        trial_days = d.pop("trialDays")

        yearly = d.pop("yearly")

        includes = d.pop("includes", UNSET)

        website_tier_option = cls(
            currency=currency,
            id=id,
            label=label,
            monthly=monthly,
            name=name,
            public=public,
            trial_days=trial_days,
            yearly=yearly,
            includes=includes,
        )

        return website_tier_option

