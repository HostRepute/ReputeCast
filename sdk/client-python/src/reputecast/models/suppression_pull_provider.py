from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="SuppressionPullProvider")



@_attrs_define
class SuppressionPullProvider:
    """ 
        Attributes:
            added (int):
            name (str):
            pulled (int):
            skipped (str | Unset): Why nothing was pulled
     """

    added: int
    name: str
    pulled: int
    skipped: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        added = self.added

        name = self.name

        pulled = self.pulled

        skipped = self.skipped


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "added": added,
            "name": name,
            "pulled": pulled,
        })
        if skipped is not UNSET:
            field_dict["skipped"] = skipped

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        added = d.pop("added")

        name = d.pop("name")

        pulled = d.pop("pulled")

        skipped = d.pop("skipped", UNSET)

        suppression_pull_provider = cls(
            added=added,
            name=name,
            pulled=pulled,
            skipped=skipped,
        )

        return suppression_pull_provider

