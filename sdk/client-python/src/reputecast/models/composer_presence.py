from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.composer_present import ComposerPresent





T = TypeVar("T", bound="ComposerPresence")



@_attrs_define
class ComposerPresence:
    """ 
        Attributes:
            others (list[ComposerPresent]):
     """

    others: list[ComposerPresent]





    def to_dict(self) -> dict[str, Any]:
        from ..models.composer_present import ComposerPresent # noqa: PLC0415
        others = []
        for others_item_data in self.others:
            others_item = others_item_data.to_dict()
            others.append(others_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "others": others,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.composer_present import ComposerPresent # noqa: PLC0415
        d = dict(src_dict)
        others = []
        _others = d.pop("others")
        for others_item_data in (_others):
            others_item = ComposerPresent.from_dict(others_item_data)



            others.append(others_item)


        composer_presence = cls(
            others=others,
        )

        return composer_presence

