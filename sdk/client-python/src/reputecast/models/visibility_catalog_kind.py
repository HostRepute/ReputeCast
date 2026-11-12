from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.operator import Operator





T = TypeVar("T", bound="VisibilityCatalogKind")



@_attrs_define
class VisibilityCatalogKind:
    """ 
        Attributes:
            group (str):
            key (str):
            ops (list[Operator]):
     """

    group: str
    key: str
    ops: list[Operator]





    def to_dict(self) -> dict[str, Any]:
        from ..models.operator import Operator # noqa: PLC0415
        group = self.group

        key = self.key

        ops = []
        for ops_item_data in self.ops:
            ops_item = ops_item_data.to_dict()
            ops.append(ops_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "group": group,
            "key": key,
            "ops": ops,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.operator import Operator # noqa: PLC0415
        d = dict(src_dict)
        group = d.pop("group")

        key = d.pop("key")

        ops = []
        _ops = d.pop("ops")
        for ops_item_data in (_ops):
            ops_item = Operator.from_dict(ops_item_data)



            ops.append(ops_item)


        visibility_catalog_kind = cls(
            group=group,
            key=key,
            ops=ops,
        )

        return visibility_catalog_kind

