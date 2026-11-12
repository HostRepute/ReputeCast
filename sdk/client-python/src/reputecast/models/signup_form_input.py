from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.signup_form_input_after_submit import SignupFormInputAfterSubmit
from ..models.signup_form_input_kind import SignupFormInputKind
from typing import cast

if TYPE_CHECKING:
  from ..models.signup_form_content import SignupFormContent
  from ..models.signup_form_field import SignupFormField
  from ..models.signup_form_spam import SignupFormSpam





T = TypeVar("T", bound="SignupFormInput")



@_attrs_define
class SignupFormInput:
    """ 
        Attributes:
            after_submit (SignupFormInputAfterSubmit):
            allowed_origins (list[str]):
            consent_text (str):
            content (SignupFormContent):
            default_list_ids (list[str]):
            double_opt_in (bool):
            fields (list[SignupFormField]):
            kind (SignupFormInputKind):
            list_choice (bool):
            list_ids (list[str]):
            name (str):
            publish (bool):
            redirect_url (str):
            spam (SignupFormSpam):
     """

    after_submit: SignupFormInputAfterSubmit
    allowed_origins: list[str]
    consent_text: str
    content: SignupFormContent
    default_list_ids: list[str]
    double_opt_in: bool
    fields: list[SignupFormField]
    kind: SignupFormInputKind
    list_choice: bool
    list_ids: list[str]
    name: str
    publish: bool
    redirect_url: str
    spam: SignupFormSpam





    def to_dict(self) -> dict[str, Any]:
        from ..models.signup_form_content import SignupFormContent # noqa: PLC0415
        from ..models.signup_form_field import SignupFormField # noqa: PLC0415
        from ..models.signup_form_spam import SignupFormSpam # noqa: PLC0415
        after_submit = self.after_submit.value

        allowed_origins = self.allowed_origins



        consent_text = self.consent_text

        content = self.content.to_dict()

        default_list_ids = self.default_list_ids



        double_opt_in = self.double_opt_in

        fields = []
        for fields_item_data in self.fields:
            fields_item = fields_item_data.to_dict()
            fields.append(fields_item)



        kind = self.kind.value

        list_choice = self.list_choice

        list_ids = self.list_ids



        name = self.name

        publish = self.publish

        redirect_url = self.redirect_url

        spam = self.spam.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "afterSubmit": after_submit,
            "allowedOrigins": allowed_origins,
            "consentText": consent_text,
            "content": content,
            "defaultListIds": default_list_ids,
            "doubleOptIn": double_opt_in,
            "fields": fields,
            "kind": kind,
            "listChoice": list_choice,
            "listIds": list_ids,
            "name": name,
            "publish": publish,
            "redirectUrl": redirect_url,
            "spam": spam,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.signup_form_content import SignupFormContent # noqa: PLC0415
        from ..models.signup_form_field import SignupFormField # noqa: PLC0415
        from ..models.signup_form_spam import SignupFormSpam # noqa: PLC0415
        d = dict(src_dict)
        after_submit = SignupFormInputAfterSubmit(d.pop("afterSubmit"))




        allowed_origins = cast(list[str], d.pop("allowedOrigins"))


        consent_text = d.pop("consentText")

        content = SignupFormContent.from_dict(d.pop("content"))




        default_list_ids = cast(list[str], d.pop("defaultListIds"))


        double_opt_in = d.pop("doubleOptIn")

        fields = []
        _fields = d.pop("fields")
        for fields_item_data in (_fields):
            fields_item = SignupFormField.from_dict(fields_item_data)



            fields.append(fields_item)


        kind = SignupFormInputKind(d.pop("kind"))




        list_choice = d.pop("listChoice")

        list_ids = cast(list[str], d.pop("listIds"))


        name = d.pop("name")

        publish = d.pop("publish")

        redirect_url = d.pop("redirectUrl")

        spam = SignupFormSpam.from_dict(d.pop("spam"))




        signup_form_input = cls(
            after_submit=after_submit,
            allowed_origins=allowed_origins,
            consent_text=consent_text,
            content=content,
            default_list_ids=default_list_ids,
            double_opt_in=double_opt_in,
            fields=fields,
            kind=kind,
            list_choice=list_choice,
            list_ids=list_ids,
            name=name,
            publish=publish,
            redirect_url=redirect_url,
            spam=spam,
        )

        return signup_form_input

