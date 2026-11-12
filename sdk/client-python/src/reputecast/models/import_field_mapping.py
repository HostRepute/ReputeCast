from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.import_field_row import ImportFieldRow





T = TypeVar("T", bound="ImportFieldMapping")



@_attrs_define
class ImportFieldMapping:
    """ 
        Attributes:
            fields (list[ImportFieldRow]):
            mapped (int):
            new (int):
            ready (bool):
            skipped (int):
     """

    fields: list[ImportFieldRow]
    mapped: int
    new: int
    ready: bool
    skipped: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.import_field_row import ImportFieldRow # noqa: PLC0415
        fields = []
        for fields_item_data in self.fields:
            fields_item = fields_item_data.to_dict()
            fields.append(fields_item)



        mapped = self.mapped

        new = self.new

        ready = self.ready

        skipped = self.skipped


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "fields": fields,
            "mapped": mapped,
            "new": new,
            "ready": ready,
            "skipped": skipped,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.import_field_row import ImportFieldRow # noqa: PLC0415
        d = dict(src_dict)
        fields = []
        _fields = d.pop("fields")
        for fields_item_data in (_fields):
            fields_item = ImportFieldRow.from_dict(fields_item_data)



            fields.append(fields_item)


        mapped = d.pop("mapped")

        new = d.pop("new")

        ready = d.pop("ready")

        skipped = d.pop("skipped")

        import_field_mapping = cls(
            fields=fields,
            mapped=mapped,
            new=new,
            ready=ready,
            skipped=skipped,
        )

        return import_field_mapping

