from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.referral_email_version import ReferralEmailVersion





T = TypeVar("T", bound="ReferralMilestoneEmailDetail")



@_attrs_define
class ReferralMilestoneEmailDetail:
    """ 
        Attributes:
            count (int):
            description (str):
            file_name (str):
            file_size (int):
            file_type (str):
            html (str):
            list_name (str):
            milestone_id (str):
            mjml (str):
            on (bool):
            preheader (str):
            program_id (str):
            revision (int):
            reward_name (str):
            reward_type (str):
            sender (str):
            sender_default (bool):
            subject (str):
            uploaded_by (str):
            versions (list[ReferralEmailVersion]):
            saved_at (datetime.datetime | Unset):
     """

    count: int
    description: str
    file_name: str
    file_size: int
    file_type: str
    html: str
    list_name: str
    milestone_id: str
    mjml: str
    on: bool
    preheader: str
    program_id: str
    revision: int
    reward_name: str
    reward_type: str
    sender: str
    sender_default: bool
    subject: str
    uploaded_by: str
    versions: list[ReferralEmailVersion]
    saved_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.referral_email_version import ReferralEmailVersion # noqa: PLC0415
        count = self.count

        description = self.description

        file_name = self.file_name

        file_size = self.file_size

        file_type = self.file_type

        html = self.html

        list_name = self.list_name

        milestone_id = self.milestone_id

        mjml = self.mjml

        on = self.on

        preheader = self.preheader

        program_id = self.program_id

        revision = self.revision

        reward_name = self.reward_name

        reward_type = self.reward_type

        sender = self.sender

        sender_default = self.sender_default

        subject = self.subject

        uploaded_by = self.uploaded_by

        versions = []
        for versions_item_data in self.versions:
            versions_item = versions_item_data.to_dict()
            versions.append(versions_item)



        saved_at: str | Unset = UNSET
        if not isinstance(self.saved_at, Unset):
            saved_at = self.saved_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "description": description,
            "fileName": file_name,
            "fileSize": file_size,
            "fileType": file_type,
            "html": html,
            "listName": list_name,
            "milestoneId": milestone_id,
            "mjml": mjml,
            "on": on,
            "preheader": preheader,
            "programId": program_id,
            "revision": revision,
            "rewardName": reward_name,
            "rewardType": reward_type,
            "sender": sender,
            "senderDefault": sender_default,
            "subject": subject,
            "uploadedBy": uploaded_by,
            "versions": versions,
        })
        if saved_at is not UNSET:
            field_dict["savedAt"] = saved_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.referral_email_version import ReferralEmailVersion # noqa: PLC0415
        d = dict(src_dict)
        count = d.pop("count")

        description = d.pop("description")

        file_name = d.pop("fileName")

        file_size = d.pop("fileSize")

        file_type = d.pop("fileType")

        html = d.pop("html")

        list_name = d.pop("listName")

        milestone_id = d.pop("milestoneId")

        mjml = d.pop("mjml")

        on = d.pop("on")

        preheader = d.pop("preheader")

        program_id = d.pop("programId")

        revision = d.pop("revision")

        reward_name = d.pop("rewardName")

        reward_type = d.pop("rewardType")

        sender = d.pop("sender")

        sender_default = d.pop("senderDefault")

        subject = d.pop("subject")

        uploaded_by = d.pop("uploadedBy")

        versions = []
        _versions = d.pop("versions")
        for versions_item_data in (_versions):
            versions_item = ReferralEmailVersion.from_dict(versions_item_data)



            versions.append(versions_item)


        _saved_at = d.pop("savedAt", UNSET)
        saved_at: datetime.datetime | Unset
        if isinstance(_saved_at,  Unset):
            saved_at = UNSET
        else:
            saved_at = datetime.datetime.fromisoformat(_saved_at)




        referral_milestone_email_detail = cls(
            count=count,
            description=description,
            file_name=file_name,
            file_size=file_size,
            file_type=file_type,
            html=html,
            list_name=list_name,
            milestone_id=milestone_id,
            mjml=mjml,
            on=on,
            preheader=preheader,
            program_id=program_id,
            revision=revision,
            reward_name=reward_name,
            reward_type=reward_type,
            sender=sender,
            sender_default=sender_default,
            subject=subject,
            uploaded_by=uploaded_by,
            versions=versions,
            saved_at=saved_at,
        )

        return referral_milestone_email_detail

