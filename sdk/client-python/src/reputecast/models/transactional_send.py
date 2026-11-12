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
  from ..models.transactional_address import TransactionalAddress
  from ..models.transactional_attachment_input import TransactionalAttachmentInput
  from ..models.transactional_send_data import TransactionalSendData
  from ..models.transactional_send_metadata import TransactionalSendMetadata





T = TypeVar("T", bound="TransactionalSend")



@_attrs_define
class TransactionalSend:
    """ 
        Attributes:
            template (str): The template's ID Example: password-reset.
            to (TransactionalAddress):
            attachments (list[TransactionalAttachmentInput] | Unset):
            bcc (list[TransactionalAddress] | Unset):
            cc (list[TransactionalAddress] | Unset):
            data (TransactionalSendData | Unset): The template's variables
            from_ (str | Unset): A verified Transactional sender, as an address or "Name <address>"; the template's sender
                when omitted
            language (str | Unset): The language version to send; the linked contact's language, then the fallback, when
                omitted Example: es.
            metadata (TransactionalSendMetadata | Unset):
            preheader (str | Unset): Overrides the template's preheader (Liquid)
            reply_to (str | Unset):
            send_at (datetime.datetime | Unset): Send later, up to 72 hours ahead
            subject (str | Unset): Overrides the template's subject (Liquid)
            tags (list[str] | Unset): Returned with every webhook event of the message
     """

    template: str
    to: TransactionalAddress
    attachments: list[TransactionalAttachmentInput] | Unset = UNSET
    bcc: list[TransactionalAddress] | Unset = UNSET
    cc: list[TransactionalAddress] | Unset = UNSET
    data: TransactionalSendData | Unset = UNSET
    from_: str | Unset = UNSET
    language: str | Unset = UNSET
    metadata: TransactionalSendMetadata | Unset = UNSET
    preheader: str | Unset = UNSET
    reply_to: str | Unset = UNSET
    send_at: datetime.datetime | Unset = UNSET
    subject: str | Unset = UNSET
    tags: list[str] | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.transactional_address import TransactionalAddress # noqa: PLC0415
        from ..models.transactional_attachment_input import TransactionalAttachmentInput # noqa: PLC0415
        from ..models.transactional_send_data import TransactionalSendData # noqa: PLC0415
        from ..models.transactional_send_metadata import TransactionalSendMetadata # noqa: PLC0415
        template = self.template

        to = self.to.to_dict()

        attachments: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.attachments, Unset):
            attachments = []
            for attachments_item_data in self.attachments:
                attachments_item = attachments_item_data.to_dict()
                attachments.append(attachments_item)



        bcc: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.bcc, Unset):
            bcc = []
            for bcc_item_data in self.bcc:
                bcc_item = bcc_item_data.to_dict()
                bcc.append(bcc_item)



        cc: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.cc, Unset):
            cc = []
            for cc_item_data in self.cc:
                cc_item = cc_item_data.to_dict()
                cc.append(cc_item)



        data: dict[str, Any] | Unset = UNSET
        if not isinstance(self.data, Unset):
            data = self.data.to_dict()

        from_ = self.from_

        language = self.language

        metadata: dict[str, Any] | Unset = UNSET
        if not isinstance(self.metadata, Unset):
            metadata = self.metadata.to_dict()

        preheader = self.preheader

        reply_to = self.reply_to

        send_at: str | Unset = UNSET
        if not isinstance(self.send_at, Unset):
            send_at = self.send_at.isoformat()

        subject = self.subject

        tags: list[str] | Unset = UNSET
        if not isinstance(self.tags, Unset):
            tags = self.tags




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "template": template,
            "to": to,
        })
        if attachments is not UNSET:
            field_dict["attachments"] = attachments
        if bcc is not UNSET:
            field_dict["bcc"] = bcc
        if cc is not UNSET:
            field_dict["cc"] = cc
        if data is not UNSET:
            field_dict["data"] = data
        if from_ is not UNSET:
            field_dict["from"] = from_
        if language is not UNSET:
            field_dict["language"] = language
        if metadata is not UNSET:
            field_dict["metadata"] = metadata
        if preheader is not UNSET:
            field_dict["preheader"] = preheader
        if reply_to is not UNSET:
            field_dict["reply_to"] = reply_to
        if send_at is not UNSET:
            field_dict["send_at"] = send_at
        if subject is not UNSET:
            field_dict["subject"] = subject
        if tags is not UNSET:
            field_dict["tags"] = tags

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transactional_address import TransactionalAddress # noqa: PLC0415
        from ..models.transactional_attachment_input import TransactionalAttachmentInput # noqa: PLC0415
        from ..models.transactional_send_data import TransactionalSendData # noqa: PLC0415
        from ..models.transactional_send_metadata import TransactionalSendMetadata # noqa: PLC0415
        d = dict(src_dict)
        template = d.pop("template")

        to = TransactionalAddress.from_dict(d.pop("to"))




        _attachments = d.pop("attachments", UNSET)
        attachments: list[TransactionalAttachmentInput] | Unset = UNSET
        if _attachments is not UNSET:
            attachments = []
            for attachments_item_data in _attachments:
                attachments_item = TransactionalAttachmentInput.from_dict(attachments_item_data)



                attachments.append(attachments_item)


        _bcc = d.pop("bcc", UNSET)
        bcc: list[TransactionalAddress] | Unset = UNSET
        if _bcc is not UNSET:
            bcc = []
            for bcc_item_data in _bcc:
                bcc_item = TransactionalAddress.from_dict(bcc_item_data)



                bcc.append(bcc_item)


        _cc = d.pop("cc", UNSET)
        cc: list[TransactionalAddress] | Unset = UNSET
        if _cc is not UNSET:
            cc = []
            for cc_item_data in _cc:
                cc_item = TransactionalAddress.from_dict(cc_item_data)



                cc.append(cc_item)


        _data = d.pop("data", UNSET)
        data: TransactionalSendData | Unset
        if isinstance(_data,  Unset):
            data = UNSET
        else:
            data = TransactionalSendData.from_dict(_data)




        from_ = d.pop("from", UNSET)

        language = d.pop("language", UNSET)

        _metadata = d.pop("metadata", UNSET)
        metadata: TransactionalSendMetadata | Unset
        if isinstance(_metadata,  Unset):
            metadata = UNSET
        else:
            metadata = TransactionalSendMetadata.from_dict(_metadata)




        preheader = d.pop("preheader", UNSET)

        reply_to = d.pop("reply_to", UNSET)

        _send_at = d.pop("send_at", UNSET)
        send_at: datetime.datetime | Unset
        if isinstance(_send_at,  Unset):
            send_at = UNSET
        else:
            send_at = datetime.datetime.fromisoformat(_send_at)




        subject = d.pop("subject", UNSET)

        tags = cast(list[str], d.pop("tags", UNSET))


        transactional_send = cls(
            template=template,
            to=to,
            attachments=attachments,
            bcc=bcc,
            cc=cc,
            data=data,
            from_=from_,
            language=language,
            metadata=metadata,
            preheader=preheader,
            reply_to=reply_to,
            send_at=send_at,
            subject=subject,
            tags=tags,
        )

        return transactional_send

