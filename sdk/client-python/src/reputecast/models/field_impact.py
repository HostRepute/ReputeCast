from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.use import Use





T = TypeVar("T", bound="FieldImpact")



@_attrs_define
class FieldImpact:
    """ 
        Attributes:
            id (str):
            key (str):
            label (str):
            uses (list[Use]):
     """

    id: str
    key: str
    label: str
    uses: list[Use]





    def to_dict(self) -> dict[str, Any]:
        from ..models.use import Use # noqa: PLC0415
        id = self.id

        key = self.key

        label = self.label

        uses = []
        for uses_item_data in self.uses:
            uses_item = uses_item_data.to_dict()
            uses.append(uses_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "key": key,
            "label": label,
            "uses": uses,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.use import Use # noqa: PLC0415
        d = dict(src_dict)
        id = d.pop("id")

        key = d.pop("key")

        label = d.pop("label")

        uses = []
        _uses = d.pop("uses")
        for uses_item_data in (_uses):
            uses_item = Use.from_dict(uses_item_data)



            uses.append(uses_item)


        field_impact = cls(
            id=id,
            key=key,
            label=label,
            uses=uses,
        )

        return field_impact

