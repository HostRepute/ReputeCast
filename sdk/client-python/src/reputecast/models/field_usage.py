from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.use import Use





T = TypeVar("T", bound="FieldUsage")



@_attrs_define
class FieldUsage:
    """ 
        Attributes:
            id (str):
            uses (list[Use]):
            filled (int | Unset):
     """

    id: str
    uses: list[Use]
    filled: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.use import Use # noqa: PLC0415
        id = self.id

        uses = []
        for uses_item_data in self.uses:
            uses_item = uses_item_data.to_dict()
            uses.append(uses_item)



        filled = self.filled


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "uses": uses,
        })
        if filled is not UNSET:
            field_dict["filled"] = filled

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.use import Use # noqa: PLC0415
        d = dict(src_dict)
        id = d.pop("id")

        uses = []
        _uses = d.pop("uses")
        for uses_item_data in (_uses):
            uses_item = Use.from_dict(uses_item_data)



            uses.append(uses_item)


        filled = d.pop("filled", UNSET)

        field_usage = cls(
            id=id,
            uses=uses,
            filled=filled,
        )

        return field_usage

