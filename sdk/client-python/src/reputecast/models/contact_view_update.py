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





T = TypeVar("T", bound="ContactViewUpdate")



@_attrs_define
class ContactViewUpdate:
    """ 
        Attributes:
            columns (list[str] | Unset):
            filters (ContactFilter | Unset):
            name (str | Unset):
     """

    columns: list[str] | Unset = UNSET
    filters: ContactFilter | Unset = UNSET
    name: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.contact_filter import ContactFilter # noqa: PLC0415
        columns: list[str] | Unset = UNSET
        if not isinstance(self.columns, Unset):
            columns = self.columns



        filters: dict[str, Any] | Unset = UNSET
        if not isinstance(self.filters, Unset):
            filters = self.filters.to_dict()

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if columns is not UNSET:
            field_dict["columns"] = columns
        if filters is not UNSET:
            field_dict["filters"] = filters
        if name is not UNSET:
            field_dict["name"] = name

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.contact_filter import ContactFilter # noqa: PLC0415
        d = dict(src_dict)
        columns = cast(list[str], d.pop("columns", UNSET))


        _filters = d.pop("filters", UNSET)
        filters: ContactFilter | Unset
        if isinstance(_filters,  Unset):
            filters = UNSET
        else:
            filters = ContactFilter.from_dict(_filters)




        name = d.pop("name", UNSET)

        contact_view_update = cls(
            columns=columns,
            filters=filters,
            name=name,
        )

        return contact_view_update

