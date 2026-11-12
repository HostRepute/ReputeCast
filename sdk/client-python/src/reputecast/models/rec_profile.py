from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.rec_image import RecImage





T = TypeVar("T", bound="RecProfile")



@_attrs_define
class RecProfile:
    """ 
        Attributes:
            description (str):
            initials (str):
            instance (str):
            list_id (str):
            list_name (str):
            list_subscribers (int):
            name (str):
            logo (RecImage | Unset):
     """

    description: str
    initials: str
    instance: str
    list_id: str
    list_name: str
    list_subscribers: int
    name: str
    logo: RecImage | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.rec_image import RecImage # noqa: PLC0415
        description = self.description

        initials = self.initials

        instance = self.instance

        list_id = self.list_id

        list_name = self.list_name

        list_subscribers = self.list_subscribers

        name = self.name

        logo: dict[str, Any] | Unset = UNSET
        if not isinstance(self.logo, Unset):
            logo = self.logo.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "description": description,
            "initials": initials,
            "instance": instance,
            "listId": list_id,
            "listName": list_name,
            "listSubscribers": list_subscribers,
            "name": name,
        })
        if logo is not UNSET:
            field_dict["logo"] = logo

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.rec_image import RecImage # noqa: PLC0415
        d = dict(src_dict)
        description = d.pop("description")

        initials = d.pop("initials")

        instance = d.pop("instance")

        list_id = d.pop("listId")

        list_name = d.pop("listName")

        list_subscribers = d.pop("listSubscribers")

        name = d.pop("name")

        _logo = d.pop("logo", UNSET)
        logo: RecImage | Unset
        if isinstance(_logo,  Unset):
            logo = UNSET
        else:
            logo = RecImage.from_dict(_logo)




        rec_profile = cls(
            description=description,
            initials=initials,
            instance=instance,
            list_id=list_id,
            list_name=list_name,
            list_subscribers=list_subscribers,
            name=name,
            logo=logo,
        )

        return rec_profile

