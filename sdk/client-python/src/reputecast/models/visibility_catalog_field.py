from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.operator import Operator





T = TypeVar("T", bound="VisibilityCatalogField")



@_attrs_define
class VisibilityCatalogField:
    """ 
        Attributes:
            key (str):
            label (str):
            ops (list[Operator]):
            options (list[str]):
            type_ (str):
     """

    key: str
    label: str
    ops: list[Operator]
    options: list[str]
    type_: str





    def to_dict(self) -> dict[str, Any]:
        from ..models.operator import Operator # noqa: PLC0415
        key = self.key

        label = self.label

        ops = []
        for ops_item_data in self.ops:
            ops_item = ops_item_data.to_dict()
            ops.append(ops_item)



        options = self.options



        type_ = self.type_


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
            "label": label,
            "ops": ops,
            "options": options,
            "type": type_,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.operator import Operator # noqa: PLC0415
        d = dict(src_dict)
        key = d.pop("key")

        label = d.pop("label")

        ops = []
        _ops = d.pop("ops")
        for ops_item_data in (_ops):
            ops_item = Operator.from_dict(ops_item_data)



            ops.append(ops_item)


        options = cast(list[str], d.pop("options"))


        type_ = d.pop("type")

        visibility_catalog_field = cls(
            key=key,
            label=label,
            ops=ops,
            options=options,
            type_=type_,
        )

        return visibility_catalog_field

