from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.earn_member_email_email import EarnMemberEmailEmail
from ..models.earn_member_email_status import EarnMemberEmailStatus
from ..types import UNSET, Unset






T = TypeVar("T", bound="EarnMemberEmail")



@_attrs_define
class EarnMemberEmail:
    """ 
        Attributes:
            draft_version (int):
            email (EarnMemberEmailEmail):
            live_version (int):
            name (str):
            status (EarnMemberEmailStatus):
            template (str):
            template_id (str | Unset):
     """

    draft_version: int
    email: EarnMemberEmailEmail
    live_version: int
    name: str
    status: EarnMemberEmailStatus
    template: str
    template_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        draft_version = self.draft_version

        email = self.email.value

        live_version = self.live_version

        name = self.name

        status = self.status.value

        template = self.template

        template_id = self.template_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "draftVersion": draft_version,
            "email": email,
            "liveVersion": live_version,
            "name": name,
            "status": status,
            "template": template,
        })
        if template_id is not UNSET:
            field_dict["templateId"] = template_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        draft_version = d.pop("draftVersion")

        email = EarnMemberEmailEmail(d.pop("email"))




        live_version = d.pop("liveVersion")

        name = d.pop("name")

        status = EarnMemberEmailStatus(d.pop("status"))




        template = d.pop("template")

        template_id = d.pop("templateId", UNSET)

        earn_member_email = cls(
            draft_version=draft_version,
            email=email,
            live_version=live_version,
            name=name,
            status=status,
            template=template,
            template_id=template_id,
        )

        return earn_member_email

