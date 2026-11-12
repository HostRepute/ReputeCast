from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.digest_filters_start_mode import DigestFiltersStartMode
from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="DigestFilters")



@_attrs_define
class DigestFilters:
    """ 
        Attributes:
            categories (list[str]):
            start_latest (int):
            start_mode (DigestFiltersStartMode):
            tags (list[str]):
            max_age_days (int | Unset):
     """

    categories: list[str]
    start_latest: int
    start_mode: DigestFiltersStartMode
    tags: list[str]
    max_age_days: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        categories = self.categories



        start_latest = self.start_latest

        start_mode = self.start_mode.value

        tags = self.tags



        max_age_days = self.max_age_days


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "categories": categories,
            "startLatest": start_latest,
            "startMode": start_mode,
            "tags": tags,
        })
        if max_age_days is not UNSET:
            field_dict["maxAgeDays"] = max_age_days

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        categories = cast(list[str], d.pop("categories"))


        start_latest = d.pop("startLatest")

        start_mode = DigestFiltersStartMode(d.pop("startMode"))




        tags = cast(list[str], d.pop("tags"))


        max_age_days = d.pop("maxAgeDays", UNSET)

        digest_filters = cls(
            categories=categories,
            start_latest=start_latest,
            start_mode=start_mode,
            tags=tags,
            max_age_days=max_age_days,
        )

        return digest_filters

