from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.push_site_prompt_status import PushSitePromptStatus
from ..models.push_site_prompt_style import PushSitePromptStyle
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="PushSitePrompt")



@_attrs_define
class PushSitePrompt:
    """ 
        Attributes:
            form_id (str):
            name (str):
            status (PushSitePromptStatus):
            style (PushSitePromptStyle):
            live_since (datetime.datetime | Unset):
     """

    form_id: str
    name: str
    status: PushSitePromptStatus
    style: PushSitePromptStyle
    live_since: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        form_id = self.form_id

        name = self.name

        status = self.status.value

        style = self.style.value

        live_since: str | Unset = UNSET
        if not isinstance(self.live_since, Unset):
            live_since = self.live_since.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "formId": form_id,
            "name": name,
            "status": status,
            "style": style,
        })
        if live_since is not UNSET:
            field_dict["liveSince"] = live_since

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        form_id = d.pop("formId")

        name = d.pop("name")

        status = PushSitePromptStatus(d.pop("status"))




        style = PushSitePromptStyle(d.pop("style"))




        _live_since = d.pop("liveSince", UNSET)
        live_since: datetime.datetime | Unset
        if isinstance(_live_since,  Unset):
            live_since = UNSET
        else:
            live_since = datetime.datetime.fromisoformat(_live_since)




        push_site_prompt = cls(
            form_id=form_id,
            name=name,
            status=status,
            style=style,
            live_since=live_since,
        )

        return push_site_prompt

