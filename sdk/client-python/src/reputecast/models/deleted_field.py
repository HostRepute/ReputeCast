from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.deleted_field_display import DeletedFieldDisplay
from ..models.deleted_field_type import DeletedFieldType
from typing import cast
import datetime






T = TypeVar("T", bound="DeletedField")



@_attrs_define
class DeletedField:
    """ 
        Attributes:
            created_at (datetime.datetime):
            deleted_at (datetime.datetime):
            display (DeletedFieldDisplay):
            id (str):
            key (str):
            label (str):
            options (list[str]):
            restore_until (datetime.datetime):
            type_ (DeletedFieldType):
     """

    created_at: datetime.datetime
    deleted_at: datetime.datetime
    display: DeletedFieldDisplay
    id: str
    key: str
    label: str
    options: list[str]
    restore_until: datetime.datetime
    type_: DeletedFieldType





    def to_dict(self) -> dict[str, Any]:
        created_at = self.created_at.isoformat()

        deleted_at = self.deleted_at.isoformat()

        display = self.display.value

        id = self.id

        key = self.key

        label = self.label

        options = self.options



        restore_until = self.restore_until.isoformat()

        type_ = self.type_.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "createdAt": created_at,
            "deletedAt": deleted_at,
            "display": display,
            "id": id,
            "key": key,
            "label": label,
            "options": options,
            "restoreUntil": restore_until,
            "type": type_,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        deleted_at = datetime.datetime.fromisoformat(d.pop("deletedAt"))




        display = DeletedFieldDisplay(d.pop("display"))




        id = d.pop("id")

        key = d.pop("key")

        label = d.pop("label")

        options = cast(list[str], d.pop("options"))


        restore_until = datetime.datetime.fromisoformat(d.pop("restoreUntil"))




        type_ = DeletedFieldType(d.pop("type"))




        deleted_field = cls(
            created_at=created_at,
            deleted_at=deleted_at,
            display=display,
            id=id,
            key=key,
            label=label,
            options=options,
            restore_until=restore_until,
            type_=type_,
        )

        return deleted_field

