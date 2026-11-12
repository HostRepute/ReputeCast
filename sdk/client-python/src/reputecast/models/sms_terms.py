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
  from ..models.sms_terms_content import SmsTermsContent





T = TypeVar("T", bound="SmsTerms")



@_attrs_define
class SmsTerms:
    """ 
        Attributes:
            can_edit (bool):
            content (SmsTermsContent):
            generated (SmsTermsContent):
            url (str):
            version (int):
            published_at (datetime.datetime | Unset):
            published_by (str | Unset):
     """

    can_edit: bool
    content: SmsTermsContent
    generated: SmsTermsContent
    url: str
    version: int
    published_at: datetime.datetime | Unset = UNSET
    published_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.sms_terms_content import SmsTermsContent # noqa: PLC0415
        can_edit = self.can_edit

        content = self.content.to_dict()

        generated = self.generated.to_dict()

        url = self.url

        version = self.version

        published_at: str | Unset = UNSET
        if not isinstance(self.published_at, Unset):
            published_at = self.published_at.isoformat()

        published_by = self.published_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canEdit": can_edit,
            "content": content,
            "generated": generated,
            "url": url,
            "version": version,
        })
        if published_at is not UNSET:
            field_dict["publishedAt"] = published_at
        if published_by is not UNSET:
            field_dict["publishedBy"] = published_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sms_terms_content import SmsTermsContent # noqa: PLC0415
        d = dict(src_dict)
        can_edit = d.pop("canEdit")

        content = SmsTermsContent.from_dict(d.pop("content"))




        generated = SmsTermsContent.from_dict(d.pop("generated"))




        url = d.pop("url")

        version = d.pop("version")

        _published_at = d.pop("publishedAt", UNSET)
        published_at: datetime.datetime | Unset
        if isinstance(_published_at,  Unset):
            published_at = UNSET
        else:
            published_at = datetime.datetime.fromisoformat(_published_at)




        published_by = d.pop("publishedBy", UNSET)

        sms_terms = cls(
            can_edit=can_edit,
            content=content,
            generated=generated,
            url=url,
            version=version,
            published_at=published_at,
            published_by=published_by,
        )

        return sms_terms

