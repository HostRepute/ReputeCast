from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.content_check_link_kind import ContentCheckLinkKind
from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="ContentCheckLink")



@_attrs_define
class ContentCheckLink:
    """ 
        Attributes:
            host (str):
            kind (ContentCheckLinkKind):
            path (list[int]):
            probed (bool):
            text (str):
            url (str):
            bot_blocked (bool | Unset):
            broken (bool | Unset):
            final (str | Unset):
            final_clean (bool | Unset):
            millis (int | Unset):
            status (int | Unset):
     """

    host: str
    kind: ContentCheckLinkKind
    path: list[int]
    probed: bool
    text: str
    url: str
    bot_blocked: bool | Unset = UNSET
    broken: bool | Unset = UNSET
    final: str | Unset = UNSET
    final_clean: bool | Unset = UNSET
    millis: int | Unset = UNSET
    status: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        host = self.host

        kind = self.kind.value

        path = self.path



        probed = self.probed

        text = self.text

        url = self.url

        bot_blocked = self.bot_blocked

        broken = self.broken

        final = self.final

        final_clean = self.final_clean

        millis = self.millis

        status = self.status


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "host": host,
            "kind": kind,
            "path": path,
            "probed": probed,
            "text": text,
            "url": url,
        })
        if bot_blocked is not UNSET:
            field_dict["botBlocked"] = bot_blocked
        if broken is not UNSET:
            field_dict["broken"] = broken
        if final is not UNSET:
            field_dict["final"] = final
        if final_clean is not UNSET:
            field_dict["finalClean"] = final_clean
        if millis is not UNSET:
            field_dict["millis"] = millis
        if status is not UNSET:
            field_dict["status"] = status

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        host = d.pop("host")

        kind = ContentCheckLinkKind(d.pop("kind"))




        path = cast(list[int], d.pop("path"))


        probed = d.pop("probed")

        text = d.pop("text")

        url = d.pop("url")

        bot_blocked = d.pop("botBlocked", UNSET)

        broken = d.pop("broken", UNSET)

        final = d.pop("final", UNSET)

        final_clean = d.pop("finalClean", UNSET)

        millis = d.pop("millis", UNSET)

        status = d.pop("status", UNSET)

        content_check_link = cls(
            host=host,
            kind=kind,
            path=path,
            probed=probed,
            text=text,
            url=url,
            bot_blocked=bot_blocked,
            broken=broken,
            final=final,
            final_clean=final_clean,
            millis=millis,
            status=status,
        )

        return content_check_link

