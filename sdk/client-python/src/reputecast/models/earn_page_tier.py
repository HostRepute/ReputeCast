from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.earn_page_tier_visibility import EarnPageTierVisibility






T = TypeVar("T", bound="EarnPageTier")



@_attrs_define
class EarnPageTier:
    """ 
        Attributes:
            id (str):
            name (str):
            shown (bool):
            trial_days (int):
            visibility (EarnPageTierVisibility):
     """

    id: str
    name: str
    shown: bool
    trial_days: int
    visibility: EarnPageTierVisibility





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        name = self.name

        shown = self.shown

        trial_days = self.trial_days

        visibility = self.visibility.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "name": name,
            "shown": shown,
            "trialDays": trial_days,
            "visibility": visibility,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        name = d.pop("name")

        shown = d.pop("shown")

        trial_days = d.pop("trialDays")

        visibility = EarnPageTierVisibility(d.pop("visibility"))




        earn_page_tier = cls(
            id=id,
            name=name,
            shown=shown,
            trial_days=trial_days,
            visibility=visibility,
        )

        return earn_page_tier

