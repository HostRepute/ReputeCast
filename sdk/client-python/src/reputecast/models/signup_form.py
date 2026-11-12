from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.signup_form_after_submit import SignupFormAfterSubmit
from ..models.signup_form_kind import SignupFormKind
from ..models.signup_form_status import SignupFormStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.signup_form_blocked_origin import SignupFormBlockedOrigin
  from ..models.signup_form_content import SignupFormContent
  from ..models.signup_form_field import SignupFormField
  from ..models.signup_form_spam import SignupFormSpam
  from ..models.signup_form_stats import SignupFormStats
  from ..models.signup_form_ur_ls import SignupFormURLs





T = TypeVar("T", bound="SignupForm")



@_attrs_define
class SignupForm:
    """ 
        Attributes:
            after_submit (SignupFormAfterSubmit):
            allowed_origins (list[str]):
            blocked_origins (list[SignupFormBlockedOrigin]):
            consent_text (str):
            consent_version (int):
            content (SignupFormContent):
            default_list_ids (list[str]): Offered lists that start checked
            double_opt_in (bool):
            fields (list[SignupFormField]):
            hosted_page (bool):
            hosted_url (str):
            id (str):
            kind (SignupFormKind):
            list_choice (bool): Visitors pick lists; off joins every offered list
            list_ids (list[str]): Lists offered on the form
            name (str):
            plain_html (str):
            public_id (str):  Example: frm_8c2e1a.
            redirect_url (str):
            slug (str):
            spam (SignupFormSpam):
            stats (SignupFormStats):
            status (SignupFormStatus):
            updated_at (datetime.datetime):
            urls (SignupFormURLs):
            waiting (int):
            published_at (datetime.datetime | Unset):
     """

    after_submit: SignupFormAfterSubmit
    allowed_origins: list[str]
    blocked_origins: list[SignupFormBlockedOrigin]
    consent_text: str
    consent_version: int
    content: SignupFormContent
    default_list_ids: list[str]
    double_opt_in: bool
    fields: list[SignupFormField]
    hosted_page: bool
    hosted_url: str
    id: str
    kind: SignupFormKind
    list_choice: bool
    list_ids: list[str]
    name: str
    plain_html: str
    public_id: str
    redirect_url: str
    slug: str
    spam: SignupFormSpam
    stats: SignupFormStats
    status: SignupFormStatus
    updated_at: datetime.datetime
    urls: SignupFormURLs
    waiting: int
    published_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.signup_form_blocked_origin import SignupFormBlockedOrigin # noqa: PLC0415
        from ..models.signup_form_content import SignupFormContent # noqa: PLC0415
        from ..models.signup_form_field import SignupFormField # noqa: PLC0415
        from ..models.signup_form_spam import SignupFormSpam # noqa: PLC0415
        from ..models.signup_form_stats import SignupFormStats # noqa: PLC0415
        from ..models.signup_form_ur_ls import SignupFormURLs # noqa: PLC0415
        after_submit = self.after_submit.value

        allowed_origins = self.allowed_origins



        blocked_origins = []
        for blocked_origins_item_data in self.blocked_origins:
            blocked_origins_item = blocked_origins_item_data.to_dict()
            blocked_origins.append(blocked_origins_item)



        consent_text = self.consent_text

        consent_version = self.consent_version

        content = self.content.to_dict()

        default_list_ids = self.default_list_ids



        double_opt_in = self.double_opt_in

        fields = []
        for fields_item_data in self.fields:
            fields_item = fields_item_data.to_dict()
            fields.append(fields_item)



        hosted_page = self.hosted_page

        hosted_url = self.hosted_url

        id = self.id

        kind = self.kind.value

        list_choice = self.list_choice

        list_ids = self.list_ids



        name = self.name

        plain_html = self.plain_html

        public_id = self.public_id

        redirect_url = self.redirect_url

        slug = self.slug

        spam = self.spam.to_dict()

        stats = self.stats.to_dict()

        status = self.status.value

        updated_at = self.updated_at.isoformat()

        urls = self.urls.to_dict()

        waiting = self.waiting

        published_at: str | Unset = UNSET
        if not isinstance(self.published_at, Unset):
            published_at = self.published_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "afterSubmit": after_submit,
            "allowedOrigins": allowed_origins,
            "blockedOrigins": blocked_origins,
            "consentText": consent_text,
            "consentVersion": consent_version,
            "content": content,
            "defaultListIds": default_list_ids,
            "doubleOptIn": double_opt_in,
            "fields": fields,
            "hostedPage": hosted_page,
            "hostedUrl": hosted_url,
            "id": id,
            "kind": kind,
            "listChoice": list_choice,
            "listIds": list_ids,
            "name": name,
            "plainHtml": plain_html,
            "publicId": public_id,
            "redirectUrl": redirect_url,
            "slug": slug,
            "spam": spam,
            "stats": stats,
            "status": status,
            "updatedAt": updated_at,
            "urls": urls,
            "waiting": waiting,
        })
        if published_at is not UNSET:
            field_dict["publishedAt"] = published_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.signup_form_blocked_origin import SignupFormBlockedOrigin # noqa: PLC0415
        from ..models.signup_form_content import SignupFormContent # noqa: PLC0415
        from ..models.signup_form_field import SignupFormField # noqa: PLC0415
        from ..models.signup_form_spam import SignupFormSpam # noqa: PLC0415
        from ..models.signup_form_stats import SignupFormStats # noqa: PLC0415
        from ..models.signup_form_ur_ls import SignupFormURLs # noqa: PLC0415
        d = dict(src_dict)
        after_submit = SignupFormAfterSubmit(d.pop("afterSubmit"))




        allowed_origins = cast(list[str], d.pop("allowedOrigins"))


        blocked_origins = []
        _blocked_origins = d.pop("blockedOrigins")
        for blocked_origins_item_data in (_blocked_origins):
            blocked_origins_item = SignupFormBlockedOrigin.from_dict(blocked_origins_item_data)



            blocked_origins.append(blocked_origins_item)


        consent_text = d.pop("consentText")

        consent_version = d.pop("consentVersion")

        content = SignupFormContent.from_dict(d.pop("content"))




        default_list_ids = cast(list[str], d.pop("defaultListIds"))


        double_opt_in = d.pop("doubleOptIn")

        fields = []
        _fields = d.pop("fields")
        for fields_item_data in (_fields):
            fields_item = SignupFormField.from_dict(fields_item_data)



            fields.append(fields_item)


        hosted_page = d.pop("hostedPage")

        hosted_url = d.pop("hostedUrl")

        id = d.pop("id")

        kind = SignupFormKind(d.pop("kind"))




        list_choice = d.pop("listChoice")

        list_ids = cast(list[str], d.pop("listIds"))


        name = d.pop("name")

        plain_html = d.pop("plainHtml")

        public_id = d.pop("publicId")

        redirect_url = d.pop("redirectUrl")

        slug = d.pop("slug")

        spam = SignupFormSpam.from_dict(d.pop("spam"))




        stats = SignupFormStats.from_dict(d.pop("stats"))




        status = SignupFormStatus(d.pop("status"))




        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))




        urls = SignupFormURLs.from_dict(d.pop("urls"))




        waiting = d.pop("waiting")

        _published_at = d.pop("publishedAt", UNSET)
        published_at: datetime.datetime | Unset
        if isinstance(_published_at,  Unset):
            published_at = UNSET
        else:
            published_at = datetime.datetime.fromisoformat(_published_at)




        signup_form = cls(
            after_submit=after_submit,
            allowed_origins=allowed_origins,
            blocked_origins=blocked_origins,
            consent_text=consent_text,
            consent_version=consent_version,
            content=content,
            default_list_ids=default_list_ids,
            double_opt_in=double_opt_in,
            fields=fields,
            hosted_page=hosted_page,
            hosted_url=hosted_url,
            id=id,
            kind=kind,
            list_choice=list_choice,
            list_ids=list_ids,
            name=name,
            plain_html=plain_html,
            public_id=public_id,
            redirect_url=redirect_url,
            slug=slug,
            spam=spam,
            stats=stats,
            status=status,
            updated_at=updated_at,
            urls=urls,
            waiting=waiting,
            published_at=published_at,
        )

        return signup_form

