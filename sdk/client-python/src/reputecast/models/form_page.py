from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.form_page_status import FormPageStatus






T = TypeVar("T", bound="FormPage")



@_attrs_define
class FormPage:
    """ 
        Attributes:
            id (str):
            live (bool):
            name (str):
            path (str):
            status (FormPageStatus):
            url (str):
     """

    id: str
    live: bool
    name: str
    path: str
    status: FormPageStatus
    url: str





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        live = self.live

        name = self.name

        path = self.path

        status = self.status.value

        url = self.url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "live": live,
            "name": name,
            "path": path,
            "status": status,
            "url": url,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        live = d.pop("live")

        name = d.pop("name")

        path = d.pop("path")

        status = FormPageStatus(d.pop("status"))




        url = d.pop("url")

        form_page = cls(
            id=id,
            live=live,
            name=name,
            path=path,
            status=status,
            url=url,
        )

        return form_page

