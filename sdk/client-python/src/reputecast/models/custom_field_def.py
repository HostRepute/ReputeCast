from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.custom_field_def_display import CustomFieldDefDisplay
from ..models.custom_field_def_type import CustomFieldDefType
from typing import cast
import datetime






T = TypeVar("T", bound="CustomFieldDef")



@_attrs_define
class CustomFieldDef:
    """ 
        Attributes:
            created_at (datetime.datetime):
            display (CustomFieldDefDisplay):
            id (str):
            key (str):
            label (str):
            options (list[str]):
            type_ (CustomFieldDefType):
     """

    created_at: datetime.datetime
    display: CustomFieldDefDisplay
    id: str
    key: str
    label: str
    options: list[str]
    type_: CustomFieldDefType





    def to_dict(self) -> dict[str, Any]:
        created_at = self.created_at.isoformat()

        display = self.display.value

        id = self.id

        key = self.key

        label = self.label

        options = self.options



        type_ = self.type_.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "createdAt": created_at,
            "display": display,
            "id": id,
            "key": key,
            "label": label,
            "options": options,
            "type": type_,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        display = CustomFieldDefDisplay(d.pop("display"))




        id = d.pop("id")

        key = d.pop("key")

        label = d.pop("label")

        options = cast(list[str], d.pop("options"))


        type_ = CustomFieldDefType(d.pop("type"))




        custom_field_def = cls(
            created_at=created_at,
            display=display,
            id=id,
            key=key,
            label=label,
            options=options,
            type_=type_,
        )

        return custom_field_def

