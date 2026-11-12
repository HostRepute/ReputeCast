from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.import_field_choice import ImportFieldChoice





T = TypeVar("T", bound="ImportSaveFieldsInput")



@_attrs_define
class ImportSaveFieldsInput:
    """ 
        Attributes:
            fields (list[ImportFieldChoice]):
     """

    fields: list[ImportFieldChoice]





    def to_dict(self) -> dict[str, Any]:
        from ..models.import_field_choice import ImportFieldChoice # noqa: PLC0415
        fields = []
        for fields_item_data in self.fields:
            fields_item = fields_item_data.to_dict()
            fields.append(fields_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "fields": fields,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.import_field_choice import ImportFieldChoice # noqa: PLC0415
        d = dict(src_dict)
        fields = []
        _fields = d.pop("fields")
        for fields_item_data in (_fields):
            fields_item = ImportFieldChoice.from_dict(fields_item_data)



            fields.append(fields_item)


        import_save_fields_input = cls(
            fields=fields,
        )

        return import_save_fields_input

