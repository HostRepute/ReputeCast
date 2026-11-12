from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.contact_filter import ContactFilter





T = TypeVar("T", bound="ContactViewInput")



@_attrs_define
class ContactViewInput:
    """ 
        Attributes:
            filters (ContactFilter):
            name (str):
            columns (list[str] | Unset):
            personal (bool | Unset):
     """

    filters: ContactFilter
    name: str
    columns: list[str] | Unset = UNSET
    personal: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.contact_filter import ContactFilter # noqa: PLC0415
        filters = self.filters.to_dict()

        name = self.name

        columns: list[str] | Unset = UNSET
        if not isinstance(self.columns, Unset):
            columns = self.columns



        personal = self.personal


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "filters": filters,
            "name": name,
        })
        if columns is not UNSET:
            field_dict["columns"] = columns
        if personal is not UNSET:
            field_dict["personal"] = personal

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.contact_filter import ContactFilter # noqa: PLC0415
        d = dict(src_dict)
        filters = ContactFilter.from_dict(d.pop("filters"))




        name = d.pop("name")

        columns = cast(list[str], d.pop("columns", UNSET))


        personal = d.pop("personal", UNSET)

        contact_view_input = cls(
            filters=filters,
            name=name,
            columns=columns,
            personal=personal,
        )

        return contact_view_input

