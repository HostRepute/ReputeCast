from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.segment_catalog_field_kind import SegmentCatalogFieldKind
from typing import cast

if TYPE_CHECKING:
  from ..models.segment_operator import SegmentOperator
  from ..models.segment_option import SegmentOption





T = TypeVar("T", bound="SegmentCatalogField")



@_attrs_define
class SegmentCatalogField:
    """ 
        Attributes:
            key (str):
            kind (SegmentCatalogFieldKind):
            label (str):
            ops (list[SegmentOperator]):
            options (list[SegmentOption]):
     """

    key: str
    kind: SegmentCatalogFieldKind
    label: str
    ops: list[SegmentOperator]
    options: list[SegmentOption]





    def to_dict(self) -> dict[str, Any]:
        from ..models.segment_operator import SegmentOperator # noqa: PLC0415
        from ..models.segment_option import SegmentOption # noqa: PLC0415
        key = self.key

        kind = self.kind.value

        label = self.label

        ops = []
        for ops_item_data in self.ops:
            ops_item = ops_item_data.to_dict()
            ops.append(ops_item)



        options = []
        for options_item_data in self.options:
            options_item = options_item_data.to_dict()
            options.append(options_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
            "kind": kind,
            "label": label,
            "ops": ops,
            "options": options,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.segment_operator import SegmentOperator # noqa: PLC0415
        from ..models.segment_option import SegmentOption # noqa: PLC0415
        d = dict(src_dict)
        key = d.pop("key")

        kind = SegmentCatalogFieldKind(d.pop("kind"))




        label = d.pop("label")

        ops = []
        _ops = d.pop("ops")
        for ops_item_data in (_ops):
            ops_item = SegmentOperator.from_dict(ops_item_data)



            ops.append(ops_item)


        options = []
        _options = d.pop("options")
        for options_item_data in (_options):
            options_item = SegmentOption.from_dict(options_item_data)



            options.append(options_item)


        segment_catalog_field = cls(
            key=key,
            kind=kind,
            label=label,
            ops=ops,
            options=options,
        )

        return segment_catalog_field

