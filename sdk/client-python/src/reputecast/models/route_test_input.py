from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.route_test_input_stream import RouteTestInputStream
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.routing_save_input import RoutingSaveInput





T = TypeVar("T", bound="RouteTestInput")



@_attrs_define
class RouteTestInput:
    """ 
        Attributes:
            email (str):
            stream (RouteTestInputStream):
            draft (RoutingSaveInput | Unset):
            list_id (str | Unset):
            sender_domain (str | Unset):
            template (str | Unset):
     """

    email: str
    stream: RouteTestInputStream
    draft: RoutingSaveInput | Unset = UNSET
    list_id: str | Unset = UNSET
    sender_domain: str | Unset = UNSET
    template: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.routing_save_input import RoutingSaveInput # noqa: PLC0415
        email = self.email

        stream = self.stream.value

        draft: dict[str, Any] | Unset = UNSET
        if not isinstance(self.draft, Unset):
            draft = self.draft.to_dict()

        list_id = self.list_id

        sender_domain = self.sender_domain

        template = self.template


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "email": email,
            "stream": stream,
        })
        if draft is not UNSET:
            field_dict["draft"] = draft
        if list_id is not UNSET:
            field_dict["listId"] = list_id
        if sender_domain is not UNSET:
            field_dict["senderDomain"] = sender_domain
        if template is not UNSET:
            field_dict["template"] = template

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.routing_save_input import RoutingSaveInput # noqa: PLC0415
        d = dict(src_dict)
        email = d.pop("email")

        stream = RouteTestInputStream(d.pop("stream"))




        _draft = d.pop("draft", UNSET)
        draft: RoutingSaveInput | Unset
        if isinstance(_draft,  Unset):
            draft = UNSET
        else:
            draft = RoutingSaveInput.from_dict(_draft)




        list_id = d.pop("listId", UNSET)

        sender_domain = d.pop("senderDomain", UNSET)

        template = d.pop("template", UNSET)

        route_test_input = cls(
            email=email,
            stream=stream,
            draft=draft,
            list_id=list_id,
            sender_domain=sender_domain,
            template=template,
        )

        return route_test_input

