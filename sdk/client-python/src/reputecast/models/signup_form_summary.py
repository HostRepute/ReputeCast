from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.signup_form_summary_kind import SignupFormSummaryKind
from ..models.signup_form_summary_status import SignupFormSummaryStatus






T = TypeVar("T", bound="SignupFormSummary")



@_attrs_define
class SignupFormSummary:
    """ 
        Attributes:
            hosted_page (bool):
            hosted_url (str):
            id (str):
            kind (SignupFormSummaryKind):
            name (str):
            public_id (str):  Example: frm_8c2e1a.
            slug (str):
            status (SignupFormSummaryStatus):
     """

    hosted_page: bool
    hosted_url: str
    id: str
    kind: SignupFormSummaryKind
    name: str
    public_id: str
    slug: str
    status: SignupFormSummaryStatus





    def to_dict(self) -> dict[str, Any]:
        hosted_page = self.hosted_page

        hosted_url = self.hosted_url

        id = self.id

        kind = self.kind.value

        name = self.name

        public_id = self.public_id

        slug = self.slug

        status = self.status.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "hostedPage": hosted_page,
            "hostedUrl": hosted_url,
            "id": id,
            "kind": kind,
            "name": name,
            "publicId": public_id,
            "slug": slug,
            "status": status,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        hosted_page = d.pop("hostedPage")

        hosted_url = d.pop("hostedUrl")

        id = d.pop("id")

        kind = SignupFormSummaryKind(d.pop("kind"))




        name = d.pop("name")

        public_id = d.pop("publicId")

        slug = d.pop("slug")

        status = SignupFormSummaryStatus(d.pop("status"))




        signup_form_summary = cls(
            hosted_page=hosted_page,
            hosted_url=hosted_url,
            id=id,
            kind=kind,
            name=name,
            public_id=public_id,
            slug=slug,
            status=status,
        )

        return signup_form_summary

