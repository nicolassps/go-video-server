### [EN]
# Video Upload Boilerplate
This project serves as a boilerplate for those who want to upload segmented videos in 10-second buffers. After the upload, a manifest file is generated with signed URLs for video access. The project supports storage in both S3 (AWS) and GCS (Google Cloud Storage), with the possibility of using both services simultaneously.

## Table of Contents
- Configuration
- Upload Endpoint
- Requests and Responses
    - POST /upload
    - GET /video/{id}
    - GET /video/{id}/manifest
- Work in Progress (WIP)
- Next Steps
- Configuration
1. Setting up the config.yaml
The config.yaml file is used to define the configuration details for S3 and GCS storages. To use one or both services, edit the file as shown below:

```yaml
storage:
  s3:
    bucket: video-store-test
    region: us-east-1
  gcs:
    project: video-store-test
    bucket: video-store-test
    region: us-east1
```

> s3: Define the bucket and AWS region where the videos will be uploaded.
> gcs: Define the project, bucket, and region in Google Cloud Storage.

2. Environment Variables
In addition to the configuration file, the following environment variables need to be set:

#### For S3 (AWS):

> AWS_ACCESS_KEY_ID
> AWS_SECRET_ACCESS_KEY

#### For GCS (Google Cloud Storage):

> GOOGLE_APPLICATION_CREDENTIALS (path to the JSON credentials file for Google)

Ensure these variables are correctly set before running the project.

### Upload Endpoint
Video uploads are handled via an HTTP POST endpoint at /upload. This endpoint performs the following:

Creates a unique ID for the video in the embedded BoltDB database.
Starts uploading the segmented video to the configured storages (S3, GCS, or both).
Each video segment is uploaded in 10-second buffers.

### Requests and Responses
> POST /upload

This endpoint is used to upload a video and initiate the segmentation process. Once the video is uploaded, an ID is generated for the video, and its metadata is stored in BoltDB.

#### Request
```bash
curl --location 'http://localhost:8080/upload' \
--form 'video=@"/path/to/video.mp4"'
```

#### Response
```json
{
    "ID": "9137de91-b5b2-4294-a95c-5e519972a5e4",
    "VideoMetadata": {
        "Width": 1080,
        "Height": 1920,
        "Name": "tiktok.mp4",
        "Duration": "234.000000"
    },
    "Status": "pending",
    "Resolutions": null
}
```

- ID: The unique identifier for the video.
- VideoMetadata: Metadata including the video width, height, name, and duration.
- Status: Current status of the video upload (initially pending).
- TotalSegments: The total number of video segments (initially 0).
- Resolutions: List of available video resolutions (initially null).

> GET /video/{id}

Retrieves the status and metadata of a previously uploaded video by its ID.

#### Request
```bash
curl --location --request GET 'http://localhost:8080/video/9137de91-b5b2-4294-a95c-5e519972a5e4'
```

#### Response
```json
{
    "ID": "9137de91-b5b2-4294-a95c-5e519972a5e4",
    "VideoMetadata": {
        "Width": 1080,
        "Height": 1920,
        "Name": "salario_tiktok.mp4",
        "Duration": "234.000000"
    },
    "Status": "complete",
    "TotalSegments": 0,
    "Resolutions": [
        {
            "Resolution": "360p",
            "Manifest": "9137de91-b5b2-4294-a95c-5e519972a5e4/manifest_360p.m3u8",
            "TotalSegments": 24,
            "Url": "https://video-store-test.s3.amazonaws.com/<VIDEO_UUID>/manifest_360p.m3u8?AMAZON_SIGNATURE",
            "UrlExpirationTime": "2024-10-20T14:18:50.3738304-03:00"
        }
    ]
}
```

- ID: The unique identifier for the video.
- VideoMetadata: Metadata such as width, height, name, and duration of the video.
- Status: The current status of the video (e.g., complete).
- TotalSegments: Number of video segments created.
- Resolutions: Available video resolutions with manifest file locations and signed URLs for playback.

> GET /video/{id}/manifest
Retrieves the signed URL for the video manifest file at a specified resolution.

#### Request
```bash
curl --location 'http://localhost:8080/video/9137de91-b5b2-4294-a95c-5e519972a5e4/manifest?resolution=360p'
```

#### Response
```json
{
    "url": "https://video-store-test.s3.amazonaws.com/9137de91-b5b2-4294-a95c-5e519972a5e4/manifest_360p.m3u8?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=AKIAW5376NXYIZ23ZG44%2F20241020%2Fus-east-1%2Fs3%2Faws4_request&X-Amz-Date=20241020T161850Z&X-Amz-Expires=3600&X-Amz-SignedHeaders=host&X-Amz-Signature=54fee5c6125d83487a055c61cb68230a82a895fdc8cd6732767638ca736cbd2b"
}
```

- url: Signed URL for the requested video manifest, valid for a limited time (e.g., 3600 seconds).

## Work in Progress (WIP)
This project is still under development. Here are some areas that are being worked on and not yet complete:

- MongoDB and DynamoDB support: The project currently uses BoltDB as the embedded database to track video upload state. However, support for distributed databases like MongoDB and DynamoDB will be added.

- Event-driven processing: In the future, video processing will be event-driven, triggered by message queues such as SQS, RabbitMQ, and Kafka.

- Telemetry and monitoring: Implementing metrics and logging to monitor the upload system's performance and health.

- Resilience and error handling: Enhancing the robustness of the upload system by improving error handling and fault tolerance.

## Next Steps

- Add MongoDB and DynamoDB support: Enabling the option to choose between embedded or distributed databases for storing upload state.

- Event-driven video processing: Implement event queues (SQS, RabbitMQ, Kafka) for asynchronous and scalable video processing.
Enhance the robustness of the upload system: Improve error handling and the overall resilience of the upload system.

## Contributions

This project is open for contributions! If you'd like to suggest improvements or report bugs, feel free to open issues or submit pull requests.

### [PT-BR]
# Video Upload Boilerplate
Este projeto serve como um boilerplate para quem deseja realizar upload segmentado de vídeos em buffers de 10 segundos. Após o upload, um arquivo manifest é criado com URLs assinadas para acesso ao vídeo. O projeto suporta armazenamento tanto no S3 (AWS) quanto no GCS (Google Cloud Storage), com a possibilidade de usar ambos os serviços simultaneamente.

## Sumário
- Configuração
- Endpoint de Upload
- Work in Progress (WIP)
- Próximos Passos
- Configuração
1. Configurando o config.yaml
O arquivo config.yaml é utilizado para definir os detalhes de configuração dos storages S3 e GCS. Para usar um ou ambos os serviços, edite o arquivo conforme o exemplo abaixo:

```yaml
storage:
  s3:
    bucket: video-store-test
    region: us-east-1
  gcs:
    project: video-store-test
    bucket: video-store-test
    region: us-east1
```

> s3: Defina o bucket e a região da AWS para onde os vídeos serão enviados.
> gcs: Defina o projeto, bucket e a região no Google Cloud Storage.

2. Variáveis de Ambiente

Além do arquivo de configuração, as seguintes variáveis de ambiente precisam ser definidas:

### Para S3 (AWS):

> AWS_ACCESS_KEY_ID
> AWS_SECRET_ACCESS_KEY

#### Para GCS (Google Cloud Storage):

> GOOGLE_APPLICATION_CREDENTIALS (caminho para o arquivo JSON com as credenciais do Google)

Certifique-se de que essas variáveis estão configuradas corretamente antes de iniciar o projeto.

## Work in Progress (WIP)
Este projeto ainda está em desenvolvimento. Aqui estão alguns pontos que estão sendo trabalhados e ainda não estão finalizados:

- Compatibilidade com MongoDB e DynamoDB: O projeto atualmente utiliza BoltDB como banco de dados embutido para rastrear o estado do upload dos vídeos. No entanto, será adicionada compatibilidade com bancos de dados distribuídos, como MongoDB e DynamoDB.

- Processamento baseado em eventos: No futuro, o processamento de vídeo será baseado em eventos disparados por filas como SQS, RabbitMQ e Kafka.

- Telemetria e monitoramento: Implementação de métricas e logs para monitorar o desempenho e a saúde do sistema de upload.

- Resiliência e tratamento de erros: Melhorar a robustez do sistema de upload, aprimorando o tratamento de erros e a tolerância a falhas.

## Próximos Passos
- Adicionar compatibilidade com MongoDB e DynamoDB: Permitindo a escolha entre bancos de dados embutidos ou distribuídos para armazenar o estado do upload.

- Processamento de vídeos baseado em eventos: Implementar filas de eventos (SQS, RabbitMQ, Kafka) para processamento de vídeos de forma assíncrona e escalável.
Melhorar a robustez do sistema de upload: Melhorar o tratamento de erros e a resiliência do sistema de upload.

## Contribuições
Este projeto está aberto a contribuições! Caso queira sugerir melhorias ou relatar bugs, sinta-se à vontade para abrir issues ou enviar pull requests.