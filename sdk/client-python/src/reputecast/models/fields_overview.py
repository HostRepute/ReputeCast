from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.deleted_field import DeletedField
  from ..models.field_usage import FieldUsage





T = TypeVar("T", bound="FieldsOverview")



@_attrs_define
class FieldsOverview:
    """ 
        Attributes:
            deleted (list[DeletedField]):
            usage (list[FieldUsage]):
     """

    deleted: list[DeletedField]
    usage: list[FieldUsage]





    def to_dict(self) -> dict[str, Any]:
        from ..models.deleted_field import DeletedField # noqa: PLC0415
        from ..models.field_usage import FieldUsage # noqa: PLC0415
        deleted = []
        for deleted_item_data in self.deleted:
            deleted_item = deleted_item_data.to_dict()
            deleted.append(deleted_item)



        usage = []
        for usage_item_data in self.usage:
            usage_item = usage_item_data.to_dict()
            usage.append(usage_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "deleted": deleted,
            "usage": usage,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.deleted_field import DeletedField # noqa: PLC0415
        from ..models.field_usage import FieldUsage # noqa: PLC0415
        d = dict(src_dict)
        deleted = []
        _deleted = d.pop("deleted")
        for deleted_item_data in (_deleted):
            deleted_item = DeletedField.from_dict(deleted_item_data)



            deleted.append(deleted_item)


        usage = []
        _usage = d.pop("usage")
        for usage_item_data in (_usage):
            usage_item = FieldUsage.from_dict(usage_item_data)



            usage.append(usage_item)


        fields_overview = cls(
            deleted=deleted,
            usage=usage,
        )

        return fields_overview

