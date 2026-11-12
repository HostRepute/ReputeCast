from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="CallTarget")



@_attrs_define
class CallTarget:
    """ 
        Attributes:
            kind (str):
            email (str | Unset):
            endpoint (str | Unset):
            event (str | Unset):
            list_ (str | Unset):
            tag (str | Unset):
            template (str | Unset):
            title (str | Unset):
            version (str | Unset):
     """

    kind: str
    email: str | Unset = UNSET
    endpoint: str | Unset = UNSET
    event: str | Unset = UNSET
    list_: str | Unset = UNSET
    tag: str | Unset = UNSET
    template: str | Unset = UNSET
    title: str | Unset = UNSET
    version: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        kind = self.kind

        email = self.email

        endpoint = self.endpoint

        event = self.event

        list_ = self.list_

        tag = self.tag

        template = self.template

        title = self.title

        version = self.version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
        })
        if email is not UNSET:
            field_dict["email"] = email
        if endpoint is not UNSET:
            field_dict["endpoint"] = endpoint
        if event is not UNSET:
            field_dict["event"] = event
        if list_ is not UNSET:
            field_dict["list"] = list_
        if tag is not UNSET:
            field_dict["tag"] = tag
        if template is not UNSET:
            field_dict["template"] = template
        if title is not UNSET:
            field_dict["title"] = title
        if version is not UNSET:
            field_dict["version"] = version

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = d.pop("kind")

        email = d.pop("email", UNSET)

        endpoint = d.pop("endpoint", UNSET)

        event = d.pop("event", UNSET)

        list_ = d.pop("list", UNSET)

        tag = d.pop("tag", UNSET)

        template = d.pop("template", UNSET)

        title = d.pop("title", UNSET)

        version = d.pop("version", UNSET)

        call_target = cls(
            kind=kind,
            email=email,
            endpoint=endpoint,
            event=event,
            list_=list_,
            tag=tag,
            template=template,
            title=title,
            version=version,
        )

        return call_target

