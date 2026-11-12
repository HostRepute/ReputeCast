from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.use import Use





T = TypeVar("T", bound="SegmentImpact")



@_attrs_define
class SegmentImpact:
    """ 
        Attributes:
            id (str):
            name (str):
            uses (list[Use]):
     """

    id: str
    name: str
    uses: list[Use]





    def to_dict(self) -> dict[str, Any]:
        from ..models.use import Use # noqa: PLC0415
        id = self.id

        name = self.name

        uses = []
        for uses_item_data in self.uses:
            uses_item = uses_item_data.to_dict()
            uses.append(uses_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "name": name,
            "uses": uses,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.use import Use # noqa: PLC0415
        d = dict(src_dict)
        id = d.pop("id")

        name = d.pop("name")

        uses = []
        _uses = d.pop("uses")
        for uses_item_data in (_uses):
            uses_item = Use.from_dict(uses_item_data)



            uses.append(uses_item)


        segment_impact = cls(
            id=id,
            name=name,
            uses=uses,
        )

        return segment_impact

