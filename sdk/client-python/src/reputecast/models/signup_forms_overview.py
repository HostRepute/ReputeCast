from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.public_branding import PublicBranding
  from ..models.public_custom_domain import PublicCustomDomain
  from ..models.public_turnstile import PublicTurnstile
  from ..models.signup_form_custom_field import SignupFormCustomField
  from ..models.signup_form_list import SignupFormList
  from ..models.signup_form_summary import SignupFormSummary





T = TypeVar("T", bound="SignupFormsOverview")



@_attrs_define
class SignupFormsOverview:
    """ 
        Attributes:
            branding (PublicBranding):
            custom_fields (list[SignupFormCustomField]):
            domain (PublicCustomDomain):
            forms (list[SignupFormSummary]):
            hosted_form_id (str):
            hosted_url (str):
            lists (list[SignupFormList]):
            own_origins (list[str]):
            system_mail (bool):
            system_provider (str):
            turnstile (PublicTurnstile):
     """

    branding: PublicBranding
    custom_fields: list[SignupFormCustomField]
    domain: PublicCustomDomain
    forms: list[SignupFormSummary]
    hosted_form_id: str
    hosted_url: str
    lists: list[SignupFormList]
    own_origins: list[str]
    system_mail: bool
    system_provider: str
    turnstile: PublicTurnstile





    def to_dict(self) -> dict[str, Any]:
        from ..models.public_branding import PublicBranding # noqa: PLC0415
        from ..models.public_custom_domain import PublicCustomDomain # noqa: PLC0415
        from ..models.public_turnstile import PublicTurnstile # noqa: PLC0415
        from ..models.signup_form_custom_field import SignupFormCustomField # noqa: PLC0415
        from ..models.signup_form_list import SignupFormList # noqa: PLC0415
        from ..models.signup_form_summary import SignupFormSummary # noqa: PLC0415
        branding = self.branding.to_dict()

        custom_fields = []
        for custom_fields_item_data in self.custom_fields:
            custom_fields_item = custom_fields_item_data.to_dict()
            custom_fields.append(custom_fields_item)



        domain = self.domain.to_dict()

        forms = []
        for forms_item_data in self.forms:
            forms_item = forms_item_data.to_dict()
            forms.append(forms_item)



        hosted_form_id = self.hosted_form_id

        hosted_url = self.hosted_url

        lists = []
        for lists_item_data in self.lists:
            lists_item = lists_item_data.to_dict()
            lists.append(lists_item)



        own_origins = self.own_origins



        system_mail = self.system_mail

        system_provider = self.system_provider

        turnstile = self.turnstile.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "branding": branding,
            "customFields": custom_fields,
            "domain": domain,
            "forms": forms,
            "hostedFormId": hosted_form_id,
            "hostedUrl": hosted_url,
            "lists": lists,
            "ownOrigins": own_origins,
            "systemMail": system_mail,
            "systemProvider": system_provider,
            "turnstile": turnstile,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.public_branding import PublicBranding # noqa: PLC0415
        from ..models.public_custom_domain import PublicCustomDomain # noqa: PLC0415
        from ..models.public_turnstile import PublicTurnstile # noqa: PLC0415
        from ..models.signup_form_custom_field import SignupFormCustomField # noqa: PLC0415
        from ..models.signup_form_list import SignupFormList # noqa: PLC0415
        from ..models.signup_form_summary import SignupFormSummary # noqa: PLC0415
        d = dict(src_dict)
        branding = PublicBranding.from_dict(d.pop("branding"))




        custom_fields = []
        _custom_fields = d.pop("customFields")
        for custom_fields_item_data in (_custom_fields):
            custom_fields_item = SignupFormCustomField.from_dict(custom_fields_item_data)



            custom_fields.append(custom_fields_item)


        domain = PublicCustomDomain.from_dict(d.pop("domain"))




        forms = []
        _forms = d.pop("forms")
        for forms_item_data in (_forms):
            forms_item = SignupFormSummary.from_dict(forms_item_data)



            forms.append(forms_item)


        hosted_form_id = d.pop("hostedFormId")

        hosted_url = d.pop("hostedUrl")

        lists = []
        _lists = d.pop("lists")
        for lists_item_data in (_lists):
            lists_item = SignupFormList.from_dict(lists_item_data)



            lists.append(lists_item)


        own_origins = cast(list[str], d.pop("ownOrigins"))


        system_mail = d.pop("systemMail")

        system_provider = d.pop("systemProvider")

        turnstile = PublicTurnstile.from_dict(d.pop("turnstile"))




        signup_forms_overview = cls(
            branding=branding,
            custom_fields=custom_fields,
            domain=domain,
            forms=forms,
            hosted_form_id=hosted_form_id,
            hosted_url=hosted_url,
            lists=lists,
            own_origins=own_origins,
            system_mail=system_mail,
            system_provider=system_provider,
            turnstile=turnstile,
        )

        return signup_forms_overview

