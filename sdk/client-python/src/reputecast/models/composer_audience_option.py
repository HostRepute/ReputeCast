from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="ComposerAudienceOption")



@_attrs_define
class ComposerAudienceOption:
    """ 
        Attributes:
            id (str):
            name (str):
            count (int | Unset):
     """

    id: str
    name: str
    count: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        name = self.name

        count = self.count


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "name": name,
        })
        if count is not UNSET:
            field_dict["count"] = count

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        name = d.pop("name")

        count = d.pop("count", UNSET)

        composer_audience_option = cls(
            id=id,
            name=name,
            count=count,
        )

        return composer_audience_option

