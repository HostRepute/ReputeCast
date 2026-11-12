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
  from ..models.transactional_sender_ref import TransactionalSenderRef
  from ..models.transactional_test_data import TransactionalTestData
  from ..models.transactional_variable import TransactionalVariable





T = TypeVar("T", bound="TransactionalVersionDetail")



@_attrs_define
class TransactionalVersionDetail:
    """ 
        Attributes:
            avatar_color (str):
            change (str):
            editor_mode (str):
            html (str):
            live (bool):
            mjml (str):
            note (str):
            plain_text (str):
            preheader (str):
            published_at (datetime.datetime):
            published_by (str):
            reply_to (str):
            size_bytes (int):
            subject (str):
            test_data (list[TransactionalTestData]):
            track_clicks (bool):
            track_opens (bool):
            variables (list[TransactionalVariable]):
            version (int):
            sender (TransactionalSenderRef | Unset):
     """

    avatar_color: str
    change: str
    editor_mode: str
    html: str
    live: bool
    mjml: str
    note: str
    plain_text: str
    preheader: str
    published_at: datetime.datetime
    published_by: str
    reply_to: str
    size_bytes: int
    subject: str
    test_data: list[TransactionalTestData]
    track_clicks: bool
    track_opens: bool
    variables: list[TransactionalVariable]
    version: int
    sender: TransactionalSenderRef | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.transactional_sender_ref import TransactionalSenderRef # noqa: PLC0415
        from ..models.transactional_test_data import TransactionalTestData # noqa: PLC0415
        from ..models.transactional_variable import TransactionalVariable # noqa: PLC0415
        avatar_color = self.avatar_color

        change = self.change

        editor_mode = self.editor_mode

        html = self.html

        live = self.live

        mjml = self.mjml

        note = self.note

        plain_text = self.plain_text

        preheader = self.preheader

        published_at = self.published_at.isoformat()

        published_by = self.published_by

        reply_to = self.reply_to

        size_bytes = self.size_bytes

        subject = self.subject

        test_data = []
        for test_data_item_data in self.test_data:
            test_data_item = test_data_item_data.to_dict()
            test_data.append(test_data_item)



        track_clicks = self.track_clicks

        track_opens = self.track_opens

        variables = []
        for variables_item_data in self.variables:
            variables_item = variables_item_data.to_dict()
            variables.append(variables_item)



        version = self.version

        sender: dict[str, Any] | Unset = UNSET
        if not isinstance(self.sender, Unset):
            sender = self.sender.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "avatarColor": avatar_color,
            "change": change,
            "editorMode": editor_mode,
            "html": html,
            "live": live,
            "mjml": mjml,
            "note": note,
            "plainText": plain_text,
            "preheader": preheader,
            "publishedAt": published_at,
            "publishedBy": published_by,
            "replyTo": reply_to,
            "sizeBytes": size_bytes,
            "subject": subject,
            "testData": test_data,
            "trackClicks": track_clicks,
            "trackOpens": track_opens,
            "variables": variables,
            "version": version,
        })
        if sender is not UNSET:
            field_dict["sender"] = sender

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transactional_sender_ref import TransactionalSenderRef # noqa: PLC0415
        from ..models.transactional_test_data import TransactionalTestData # noqa: PLC0415
        from ..models.transactional_variable import TransactionalVariable # noqa: PLC0415
        d = dict(src_dict)
        avatar_color = d.pop("avatarColor")

        change = d.pop("change")

        editor_mode = d.pop("editorMode")

        html = d.pop("html")

        live = d.pop("live")

        mjml = d.pop("mjml")

        note = d.pop("note")

        plain_text = d.pop("plainText")

        preheader = d.pop("preheader")

        published_at = datetime.datetime.fromisoformat(d.pop("publishedAt"))




        published_by = d.pop("publishedBy")

        reply_to = d.pop("replyTo")

        size_bytes = d.pop("sizeBytes")

        subject = d.pop("subject")

        test_data = []
        _test_data = d.pop("testData")
        for test_data_item_data in (_test_data):
            test_data_item = TransactionalTestData.from_dict(test_data_item_data)



            test_data.append(test_data_item)


        track_clicks = d.pop("trackClicks")

        track_opens = d.pop("trackOpens")

        variables = []
        _variables = d.pop("variables")
        for variables_item_data in (_variables):
            variables_item = TransactionalVariable.from_dict(variables_item_data)



            variables.append(variables_item)


        version = d.pop("version")

        _sender = d.pop("sender", UNSET)
        sender: TransactionalSenderRef | Unset
        if isinstance(_sender,  Unset):
            sender = UNSET
        else:
            sender = TransactionalSenderRef.from_dict(_sender)




        transactional_version_detail = cls(
            avatar_color=avatar_color,
            change=change,
            editor_mode=editor_mode,
            html=html,
            live=live,
            mjml=mjml,
            note=note,
            plain_text=plain_text,
            preheader=preheader,
            published_at=published_at,
            published_by=published_by,
            reply_to=reply_to,
            size_bytes=size_bytes,
            subject=subject,
            test_data=test_data,
            track_clicks=track_clicks,
            track_opens=track_opens,
            variables=variables,
            version=version,
            sender=sender,
        )

        return transactional_version_detail

