from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.segment_catalog_field import SegmentCatalogField





T = TypeVar("T", bound="SegmentCatalog")



@_attrs_define
class SegmentCatalog:
    """ 
        Attributes:
            fields (list[SegmentCatalogField]):
            sql_enabled (bool):
            views (list[str]):
     """

    fields: list[SegmentCatalogField]
    sql_enabled: bool
    views: list[str]





    def to_dict(self) -> dict[str, Any]:
        from ..models.segment_catalog_field import SegmentCatalogField # noqa: PLC0415
        fields = []
        for fields_item_data in self.fields:
            fields_item = fields_item_data.to_dict()
            fields.append(fields_item)



        sql_enabled = self.sql_enabled

        views = self.views




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "fields": fields,
            "sqlEnabled": sql_enabled,
            "views": views,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.segment_catalog_field import SegmentCatalogField # noqa: PLC0415
        d = dict(src_dict)
        fields = []
        _fields = d.pop("fields")
        for fields_item_data in (_fields):
            fields_item = SegmentCatalogField.from_dict(fields_item_data)



            fields.append(fields_item)


        sql_enabled = d.pop("sqlEnabled")

        views = cast(list[str], d.pop("views"))


        segment_catalog = cls(
            fields=fields,
            sql_enabled=sql_enabled,
            views=views,
        )

        return segment_catalog

