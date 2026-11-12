from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.render_result_status import RenderResultStatus
from ..types import UNSET, Unset






T = TypeVar("T", bound="RenderResult")



@_attrs_define
class RenderResult:
    """ 
        Attributes:
            client (str):
            name (str):
            status (RenderResultStatus):
            image_url (str | Unset): The full screenshot, hosted by the service
            thumb_url (str | Unset):
     """

    client: str
    name: str
    status: RenderResultStatus
    image_url: str | Unset = UNSET
    thumb_url: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        client = self.client

        name = self.name

        status = self.status.value

        image_url = self.image_url

        thumb_url = self.thumb_url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "client": client,
            "name": name,
            "status": status,
        })
        if image_url is not UNSET:
            field_dict["imageUrl"] = image_url
        if thumb_url is not UNSET:
            field_dict["thumbUrl"] = thumb_url

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        client = d.pop("client")

        name = d.pop("name")

        status = RenderResultStatus(d.pop("status"))




        image_url = d.pop("imageUrl", UNSET)

        thumb_url = d.pop("thumbUrl", UNSET)

        render_result = cls(
            client=client,
            name=name,
            status=status,
            image_url=image_url,
            thumb_url=thumb_url,
        )

        return render_result

