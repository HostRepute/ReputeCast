from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.pages import Pages
  from ..models.public_branding import PublicBranding
  from ..models.public_custom_domain import PublicCustomDomain
  from ..models.public_page_list import PublicPageList
  from ..models.public_page_ur_ls import PublicPageURLs
  from ..models.public_turnstile import PublicTurnstile
  from ..models.signup_form_summary import SignupFormSummary





T = TypeVar("T", bound="PublicPagesSettings")



@_attrs_define
class PublicPagesSettings:
    """ 
        Attributes:
            branding (PublicBranding):
            domain (PublicCustomDomain):
            forms (list[SignupFormSummary]):
            hosted_form_id (str):
            lists (list[PublicPageList]):
            pages (Pages):
            sender_email (str):
            turnstile (PublicTurnstile):
            updated_at (datetime.datetime):
            urls (PublicPageURLs):
     """

    branding: PublicBranding
    domain: PublicCustomDomain
    forms: list[SignupFormSummary]
    hosted_form_id: str
    lists: list[PublicPageList]
    pages: Pages
    sender_email: str
    turnstile: PublicTurnstile
    updated_at: datetime.datetime
    urls: PublicPageURLs





    def to_dict(self) -> dict[str, Any]:
        from ..models.pages import Pages # noqa: PLC0415
        from ..models.public_branding import PublicBranding # noqa: PLC0415
        from ..models.public_custom_domain import PublicCustomDomain # noqa: PLC0415
        from ..models.public_page_list import PublicPageList # noqa: PLC0415
        from ..models.public_page_ur_ls import PublicPageURLs # noqa: PLC0415
        from ..models.public_turnstile import PublicTurnstile # noqa: PLC0415
        from ..models.signup_form_summary import SignupFormSummary # noqa: PLC0415
        branding = self.branding.to_dict()

        domain = self.domain.to_dict()

        forms = []
        for forms_item_data in self.forms:
            forms_item = forms_item_data.to_dict()
            forms.append(forms_item)



        hosted_form_id = self.hosted_form_id

        lists = []
        for lists_item_data in self.lists:
            lists_item = lists_item_data.to_dict()
            lists.append(lists_item)



        pages = self.pages.to_dict()

        sender_email = self.sender_email

        turnstile = self.turnstile.to_dict()

        updated_at = self.updated_at.isoformat()

        urls = self.urls.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "branding": branding,
            "domain": domain,
            "forms": forms,
            "hostedFormId": hosted_form_id,
            "lists": lists,
            "pages": pages,
            "senderEmail": sender_email,
            "turnstile": turnstile,
            "updatedAt": updated_at,
            "urls": urls,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.pages import Pages # noqa: PLC0415
        from ..models.public_branding import PublicBranding # noqa: PLC0415
        from ..models.public_custom_domain import PublicCustomDomain # noqa: PLC0415
        from ..models.public_page_list import PublicPageList # noqa: PLC0415
        from ..models.public_page_ur_ls import PublicPageURLs # noqa: PLC0415
        from ..models.public_turnstile import PublicTurnstile # noqa: PLC0415
        from ..models.signup_form_summary import SignupFormSummary # noqa: PLC0415
        d = dict(src_dict)
        branding = PublicBranding.from_dict(d.pop("branding"))




        domain = PublicCustomDomain.from_dict(d.pop("domain"))




        forms = []
        _forms = d.pop("forms")
        for forms_item_data in (_forms):
            forms_item = SignupFormSummary.from_dict(forms_item_data)



            forms.append(forms_item)


        hosted_form_id = d.pop("hostedFormId")

        lists = []
        _lists = d.pop("lists")
        for lists_item_data in (_lists):
            lists_item = PublicPageList.from_dict(lists_item_data)



            lists.append(lists_item)


        pages = Pages.from_dict(d.pop("pages"))




        sender_email = d.pop("senderEmail")

        turnstile = PublicTurnstile.from_dict(d.pop("turnstile"))




        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))




        urls = PublicPageURLs.from_dict(d.pop("urls"))




        public_pages_settings = cls(
            branding=branding,
            domain=domain,
            forms=forms,
            hosted_form_id=hosted_form_id,
            lists=lists,
            pages=pages,
            sender_email=sender_email,
            turnstile=turnstile,
            updated_at=updated_at,
            urls=urls,
        )

        return public_pages_settings

